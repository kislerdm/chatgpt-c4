// Package ciam to authN/Z users
package ciam

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"time"

	"github.com/kislerdm/diagramastext/server/core/internal/utils"
)

// Client defines the CIAM client.
type Client interface {
	// SigninAnonym executes anonym's authentication flow.
	SigninAnonym(ctx context.Context, fingerprint string) (Tokens, error)

	// SigninUser executes user's authentication flow.
	SigninUser(ctx context.Context, email, fingerprint string) (identityToken JWT, err error)

	// IssueTokensAfterSecretConfirmation validates user's confirmation.
	// The method requires successful invocation of SigninUser and a feedback from the user.
	IssueTokensAfterSecretConfirmation(ctx context.Context, identityToken, secret string) (Tokens, error)

	// RefreshTokens refreshes access token given the refresh token.
	RefreshTokens(ctx context.Context, refreshToken string) (Tokens, error)

	// ParseAndValidateToken validates JWT.
	ParseAndValidateToken(ctx context.Context, token string) (JWT, error)
}

type Tokens struct {
	id      JWT
	refresh JWT
	access  JWT
}

func (t Tokens) Serialize() ([]byte, error) {
	var (
		temp struct {
			ID      *string `json:"id,omitempty"`
			Refresh *string `json:"refresh,omitempty"`
			Access  *string `json:"access,omitempty"`
		}
		err error
	)
	sID, err := t.id.String()
	if err != nil {
		return nil, err
	}
	temp.ID = &sID

	sRefresh, err := t.refresh.String()
	if err != nil {
		return nil, err
	}
	temp.Refresh = &sRefresh

	sAccess, err := t.access.String()
	if err != nil {
		return nil, err
	}
	temp.Access = &sAccess

	return json.Marshal(temp)
}

// NewClient initializes the CIAM client.
func NewClient(clientRepository RepositoryCIAM, clientKMS TokenSigningClient, clientEmail SMTPClient) Client {
	return &client{
		clientRepository: clientRepository,
		clientKMS:        clientKMS,
		clientEmail:      clientEmail,
	}
}

type client struct {
	clientRepository RepositoryCIAM
	clientKMS        TokenSigningClient
	clientEmail      SMTPClient
}

// SigninAnonym executes anonym's authentication flow:
//
//	Fingerprint found in DB -> No  -> Create \
//							-> Yes ->  --	-> Generate refresh and access JWT -> Return generates JWT.
func (c client) SigninAnonym(ctx context.Context, fingerprint string) (Tokens, error) {
	if fingerprint == "" {
		return Tokens{}, errors.New("fingerprint must be provided")
	}

	var (
		userID string
		err    error
	)

	userID, isActive, err := c.clientRepository.LookupUserByFingerprint(ctx, fingerprint)
	if err != nil {
		return Tokens{}, err
	}

	if userID != "" && !isActive {
		return Tokens{}, errors.New("user was deactivated")
	}

	if userID == "" {
		userID = utils.NewUUID()
		if err := c.clientRepository.CreateUser(ctx, userID, "", fingerprint, true); err != nil {
			return Tokens{}, err
		}
	}

	return c.issueTokens(ctx, userID, "", fingerprint, false)
}

// SigninUser executes user's authentication flow:
//
//	Email found in DB -> No  -> Create \
//			 	   	  -> Yes ->	--	  -> Generate secret and id JWT -> Send secret to email -> Return id JWT.
func (c client) SigninUser(ctx context.Context, email, fingerprint string) (JWT, error) {
	if email == "" {
		return nil, errors.New("email must be provided")
	}

	const defaultExpirationSecret = 10 * time.Minute

	var (
		userID     string
		err        error
		newIDToken = func(userID, email, fingerprint string, iat time.Time) (JWT, error) {
			return NewIDToken(
				userID, email, fingerprint, false, 0, WithCustomIat(iat), WithSignature(
					func(signingString string) (signature string, alg string, err error) {
						return c.clientKMS.Sign(ctx, signature)
					},
				),
			)
		}
	)

	userID, isActive, err := c.clientRepository.LookupUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	switch userID == "" {
	case true:
		userID = utils.NewUUID()
		if err := c.clientRepository.CreateUser(ctx, userID, email, fingerprint, false); err != nil {
			return nil, err
		}
	default:
		if !isActive {
			return nil, errors.New("user was deactivated")
		}
		found, _, iat, err := c.clientRepository.ReadOneTimeSecret(ctx, userID)
		if err != nil {
			return nil, err
		}
		if found && iat.Add(defaultExpirationSecret).After(time.Now().UTC()) {
			return newIDToken(userID, email, fingerprint, iat)
		}
	}

	secret := generateOnetimeSecret()
	iat := time.Now().UTC()

	if err := c.clientEmail.SendSignInEmail(email, secret); err != nil {
		return nil, err
	}
	if err := c.clientRepository.WriteOneTimeSecret(ctx, userID, secret, iat); err != nil {
		return nil, err
	}
	return newIDToken(userID, email, fingerprint, iat)
}

func (c client) IssueTokensAfterSecretConfirmation(ctx context.Context, identityToken, secret string) (Tokens, error) {
	t, err := ParseToken(identityToken)
	if err != nil {
		return Tokens{}, err
	}
	if err := t.Validate(
		func(signingString, signature string) error {
			return c.clientKMS.Verify(ctx, signingString, signature)
		},
	); err != nil {
		return Tokens{}, err
	}

	found, secretRef, _, err := c.clientRepository.ReadOneTimeSecret(ctx, t.UserID())
	if err != nil {
		return Tokens{}, err
	}

	if !found {
		return Tokens{}, errors.New("no secret was sent")
	}

	if secret != secretRef {
		return Tokens{}, errors.New("secret is wrong")
	}

	if err := c.clientRepository.UpdateUserSetEmailVerified(ctx, t.UserID()); err != nil {
		return Tokens{}, err
	}

	_ = c.clientRepository.DeleteOneTimeSecret(ctx, t.UserID())

	return c.issueTokens(ctx, t.UserID(), t.UserEmail(), t.UserDeviceFingerprint(), true)
}

func (c client) issueTokens(ctx context.Context, userID, email, fingerprint string, emailVerified bool) (
	Tokens, error,
) {
	iat := time.Now().UTC()
	opts := []OptFn{
		WithCustomIat(iat), WithSignature(
			func(signingString string) (signature string, alg string, err error) {
				return c.clientKMS.Sign(ctx, signature)
			},
		),
	}
	idToken, err := NewIDToken(userID, email, fingerprint, emailVerified, 0, opts...)
	if err != nil {
		return Tokens{}, err
	}
	accessToken, err := NewAccessToken(userID, emailVerified, opts...)
	if err != nil {
		return Tokens{}, err
	}
	refreshToken, err := NewRefreshToken(userID, opts...)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{
		id:      idToken,
		refresh: refreshToken,
		access:  accessToken,
	}, nil
}

func (c client) ParseAndValidateToken(ctx context.Context, token string) (JWT, error) {
	t, err := ParseToken(token)
	if err != nil {
		return nil, err
	}
	if err := t.Validate(
		func(signingString, signature string) error {
			return c.clientKMS.Verify(ctx, signingString, signature)
		},
	); err != nil {
		return nil, err
	}
	return t, nil
}

func (c client) RefreshTokens(ctx context.Context, refreshToken string) (Tokens, error) {
	t, err := ParseToken(refreshToken)
	if err != nil {
		return Tokens{}, err
	}
	if err := t.Validate(
		func(signingString, signature string) error {
			return c.clientKMS.Verify(ctx, signingString, signature)
		},
	); err != nil {
		return Tokens{}, err
	}
	found, isActive, emailVerified, email, fingerprint, err := c.clientRepository.ReadUser(ctx, t.UserID())
	if err != nil {
		return Tokens{}, err
	}
	if !found {
		return Tokens{}, errors.New("user not found")
	}
	if !isActive {
		return Tokens{}, errors.New("user was deactivated")
	}
	if email != "" && !emailVerified {
		return Tokens{}, errors.New("user's email was not verified yet")
	}
	return c.issueTokens(ctx, t.UserID(), email, fingerprint, emailVerified)
}

func generateOnetimeSecret() string {
	const (
		charset = "0123456789abcdef"
		length  = 6
	)
	var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	var b = make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

type MockCIAMClient struct {
	Err                        error
	UserID, Email, Fingerprint string
	tokens                     Tokens
}

func (m *MockCIAMClient) Tokens() Tokens {
	return m.tokens
}

func (m *MockCIAMClient) output() (Tokens, error) {
	if m.Err != nil {
		return Tokens{}, m.Err
	}

	userID := m.UserID
	if userID == "" {
		userID = utils.NewUUID()
	}

	emailVerified := false
	if m.Email != "" {
		emailVerified = true
	}

	iat := time.Now().UTC()

	acc, _ := NewAccessToken(userID, emailVerified, WithCustomIat(iat))
	ref, _ := NewRefreshToken(userID, WithCustomIat(iat))
	id, _ := NewIDToken(userID, m.Email, m.Fingerprint, emailVerified, 0, WithCustomIat(iat))

	m.tokens = Tokens{
		id:      id,
		refresh: ref,
		access:  acc,
	}

	return m.Tokens(), nil
}

func (m *MockCIAMClient) SigninAnonym(_ context.Context, _ string) (Tokens, error) {
	return m.output()
}

func (m *MockCIAMClient) SigninUser(_ context.Context, _, _ string) (identityToken JWT, err error) {
	t, err := m.output()
	if err != nil {
		return nil, err
	}
	return t.id, nil
}

func (m *MockCIAMClient) IssueTokensAfterSecretConfirmation(_ context.Context, _, _ string) (
	Tokens, error,
) {
	return m.output()
}

func (m *MockCIAMClient) RefreshTokens(_ context.Context, _ string) (Tokens, error) {
	return m.output()
}

func (m *MockCIAMClient) ParseAndValidateToken(_ context.Context, _ string) (JWT, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	// FIXME: make stateless method
	return m.tokens.access, nil
}

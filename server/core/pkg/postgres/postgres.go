package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// Config configuration of the postgres Client.
type Config struct {
	DBHost          string `json:"db_host"`
	DBName          string `json:"db_name"`
	DBUser          string `json:"db_user"`
	DBPassword      string `json:"db_password"`
	TablePrompt     string `json:"table_prompt,omitempty"`
	TablePrediction string `json:"table_prediction,omitempty"`
	SSLMode         string `json:"ssl_mode"`
}

func (cfg Config) Validate() error {
	if cfg.DBHost == "" {
		return errors.New("host must be provided")
	}
	if cfg.DBName == "" {
		return errors.New("dbname must be provided")
	}
	if cfg.DBUser == "" {
		return errors.New("user must be provided")
	}
	if cfg.TablePrompt == "" {
		return errors.New("table_prompt must be provided")
	}
	if cfg.TablePrediction == "" {
		return errors.New("table_prediction must be provided")
	}
	return validateSSLMode(cfg.SSLMode)
}

func validateSSLMode(mode string) error {
	switch mode {
	case "verify-full", "disable", "":
		return nil
	default:
		return errors.New("ssl mode " + mode + " is not supported")
	}
}

// NewPostgresClient initiates the postgres Client.
func NewPostgresClient(ctx context.Context, cfg Config) (
	*Client, error,
) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	connStr := "user=" + cfg.DBUser +
		" dbname=" + cfg.DBName +
		host(cfg.DBHost)

	if cfg.DBPassword != "" {
		connStr += " password=" + cfg.DBPassword
	}

	if cfg.SSLMode != "" {
		connStr += " sslmode=" + cfg.SSLMode
	}

	var db dbClient
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if cfg.DBHost == "mock" {
		db = mockDbClient{}
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return &Client{
		c:                         db,
		tableWritePrompt:          cfg.TablePrompt,
		tableWriteModelPrediction: cfg.TablePrediction,
	}, nil
}

func host(host string) string {
	hostList := strings.SplitN(host, ":", 2)
	if len(hostList) == 2 {
		if _, err := strconv.ParseUint(hostList[1], 10, 32); err == nil {
			return " host=" + hostList[0] + " port=" + hostList[1]
		}
	}
	return " host=" + host
}

type Client struct {
	c                         dbClient
	tableWritePrompt          string
	tableWriteModelPrediction string
}

func (c Client) Close(_ context.Context) error {
	return c.c.Close()
}

func (c Client) WriteInputPrompt(ctx context.Context, requestID, userID, prompt string) error {
	if requestID == "" {
		return errors.New("request_id is required")
	}
	if prompt == "" {
		return errors.New("prompt is required")
	}
	_, err := c.c.ExecContext(
		ctx, `INSERT INTO `+c.tableWritePrompt+
			` (request_id, user_id, prompt, timestamp) VALUES ($1, $2, $3, $4)`,
		requestID,
		userID,
		prompt,
		time.Now().UTC(),
	)
	return err
}

func (c Client) WriteModelResult(
	ctx context.Context, requestID, userID, prediction, model string, usageTokensPrompt, usageTokensCompletions uint16,
) error {
	if requestID == "" {
		return errors.New("request_id is required")
	}
	if prediction == "" {
		return errors.New("response is required")
	}
	if model == "" {
		return errors.New("model is required")
	}
	_, err := c.c.ExecContext(
		ctx, `INSERT INTO `+c.tableWriteModelPrediction+
			` (request_id, user_id, response, timestamp, model, prompt_tokens, completion_tokens) 
VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		requestID,
		userID,
		prediction,
		time.Now().UTC(),
		model,
		usageTokensPrompt,
		usageTokensCompletions,
	)
	return err
}

type mockDbClient struct {
	err error
}

func (m mockDbClient) Close() error {
	return m.err
}

func (m mockDbClient) PingContext(_ context.Context) error {
	return m.err
}

func (m mockDbClient) ExecContext(_ context.Context, _ string, _ ...any) (sql.Result, error) {
	return nil, m.err
}

type dbClient interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	PingContext(ctx context.Context) error
	Close() error
}

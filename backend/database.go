package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Database struct {
	SQL             *sql.DB
	Available       bool
	Driver          string
	DSN             string
	MigrationsDir   string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type AppliedMigration struct {
	Version   string    `json:"version"`
	AppliedAt time.Time `json:"appliedAt"`
}

func initDatabase(ctx context.Context) *Database {
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	cfg := &Database{
		Driver:          "pgx",
		DSN:             dsn,
		MigrationsDir:   envOrDefault("DB_MIGRATIONS_DIR", "./migrations"),
		MaxOpenConns:    envIntOrDefault("POSTGRES_MAX_OPEN_CONNS", 10),
		MaxIdleConns:    envIntOrDefault("POSTGRES_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: envDurationOrDefault("POSTGRES_CONN_MAX_LIFETIME", 30*time.Minute),
	}

	if dsn == "" {
		log.Printf("database unavailable, DATABASE_URL is empty; staying on in-memory/redis persistence")
		return cfg
	}

	db, err := sql.Open(cfg.Driver, dsn)
	if err != nil {
		log.Printf("database unavailable, open failed: %v", err)
		return cfg
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		log.Printf("database unavailable, ping failed: %v", err)
		_ = db.Close()
		return cfg
	}

	cfg.SQL = db
	cfg.Available = true
	return cfg
}

func (d *Database) Close() error {
	if d == nil || d.SQL == nil {
		return nil
	}
	return d.SQL.Close()
}

func (d *Database) MigrationFiles() ([]string, error) {
	if d == nil {
		return nil, errors.New("database config is nil")
	}

	absDir, err := filepath.Abs(d.MigrationsDir)
	if err != nil {
		return nil, err
	}
	matches, err := filepath.Glob(filepath.Join(absDir, "*.sql"))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}

func (d *Database) EnsureSchemaMigrations(ctx context.Context) error {
	if d == nil || !d.Available || d.SQL == nil {
		return nil
	}

	_, err := d.SQL.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`)
	return err
}

func (d *Database) AppliedMigrations(ctx context.Context) ([]AppliedMigration, error) {
	if d == nil || !d.Available || d.SQL == nil {
		return nil, nil
	}
	if err := d.EnsureSchemaMigrations(ctx); err != nil {
		return nil, err
	}

	rows, err := d.SQL.QueryContext(ctx, `SELECT version, applied_at FROM schema_migrations ORDER BY version ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]AppliedMigration, 0)
	for rows.Next() {
		var item AppliedMigration
		if err := rows.Scan(&item.Version, &item.AppliedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (d *Database) ApplyMigrations(ctx context.Context) error {
	if d == nil || !d.Available || d.SQL == nil {
		return nil
	}
	if err := d.EnsureSchemaMigrations(ctx); err != nil {
		return err
	}

	files, err := d.MigrationFiles()
	if err != nil {
		return err
	}

	applied, err := d.AppliedMigrations(ctx)
	if err != nil {
		return err
	}
	appliedSet := make(map[string]struct{}, len(applied))
	for _, item := range applied {
		appliedSet[item.Version] = struct{}{}
	}

	for _, file := range files {
		version := filepath.Base(file)
		if _, exists := appliedSet[version]; exists {
			continue
		}

		raw, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}

		tx, err := d.SQL.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx, string(raw)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version) VALUES ($1)`, version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", version, err)
		}
		log.Printf("applied migration %s", version)
	}

	return nil
}

func envIntOrDefault(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func envDurationOrDefault(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}

type UserStore interface {
	Create(context.Context, CreateUserRequest) (SocialUser, error)
	GetByID(context.Context, string) (*SocialUser, error)
	SaveAuthAccount(context.Context, Account) error
	FindAuthAccountByWallet(context.Context, string) (Account, bool, error)
	ListAuthAccounts(context.Context) ([]Account, error)
}

type SocialStore interface {
	CreatePost(context.Context, *SocialService, CreatePostRequest) (SocialPost, error)
	CreateMedia(context.Context, *SocialService, CreateMediaRequest) (MediaAsset, error)
	FollowUser(context.Context, *SocialService, string, string) error
	UnfollowUser(context.Context, string, string) error
	CreateConversation(context.Context, *SocialService, string, CreateConversationRequest) (Conversation, error)
	AddMessage(context.Context, *SocialService, string, CreateMessageRequest) (Conversation, error)
	ListUsers(context.Context, *SocialService) ([]SocialUser, error)
	GetUser(context.Context, *SocialService, string) (*SocialUser, error)
	UpdateUser(context.Context, *SocialService, string, UpdateUserRequest) (SocialUser, error)
	ListMedia(context.Context, *SocialService, int) ([]MediaAsset, error)
	Feed(context.Context, *SocialService, int) ([]SocialPost, error)
	FeedMine(context.Context, *SocialService, int, string) ([]SocialPost, error)
	GetPost(context.Context, *SocialService, string) (*SocialPost, error)
	GetPostThread(context.Context, *SocialService, string, int) (*PostThread, error)
	ListReplies(context.Context, *SocialService, string, int) ([]SocialPost, error)
	ListConversations(context.Context, *SocialService, int, string) ([]Conversation, error)
	GetConversation(context.Context, *SocialService, string) (*Conversation, error)
	Bootstrap(context.Context, *SocialService, int, string, bool) (BootstrapPayload, error)
	VotePoll(context.Context, *SocialService, string, string, []int) (SocialPost, error)
}

type Store struct {
	Users  UserStore
	Social SocialStore
}

type MemoryUserStore struct {
	social *SocialService
	auth   *AuthService
}

type MemorySocialStore struct {
	social *SocialService
}

func NewStore(db *Database, social *SocialService, auth *AuthService) *Store {
	if db != nil && db.Available && db.SQL != nil {
		pg := &PostgresUserStore{db: db.SQL}
		return &Store{Users: pg, Social: &PostgresSocialStore{db: db.SQL}}
	}
	return &Store{
		Users:  &MemoryUserStore{social: social, auth: auth},
		Social: &MemorySocialStore{social: social},
	}
}

func (s *MemoryUserStore) Create(ctx context.Context, req CreateUserRequest) (SocialUser, error) {
	_ = ctx
	return s.social.CreateUser(req)
}

func (s *MemoryUserStore) GetByID(ctx context.Context, id string) (*SocialUser, error) {
	_ = ctx
	return s.social.GetUser(id)
}

func (s *MemoryUserStore) SaveAuthAccount(ctx context.Context, account Account) error {
	_ = ctx
	s.auth.mu.Lock()
	defer s.auth.mu.Unlock()
	s.auth.accounts[account.ID] = account
	s.auth.persistAccountsLocked()
	return nil
}

func (s *MemoryUserStore) FindAuthAccountByWallet(ctx context.Context, wallet string) (Account, bool, error) {
	_ = ctx
	s.auth.mu.Lock()
	defer s.auth.mu.Unlock()
	account, ok := s.auth.findAccountByWalletNoLock(wallet)
	return account, ok, nil
}

func (s *MemoryUserStore) ListAuthAccounts(ctx context.Context) ([]Account, error) {
	_ = ctx
	s.auth.mu.Lock()
	defer s.auth.mu.Unlock()
	accounts := make([]Account, 0, len(s.auth.accounts))
	for _, account := range s.auth.accounts {
		accounts = append(accounts, account)
	}
	return accounts, nil
}

func (s *MemorySocialStore) CreatePost(ctx context.Context, social *SocialService, req CreatePostRequest) (SocialPost, error) {
	_ = ctx
	_ = social
	return s.social.CreatePost(req)
}

func (s *MemorySocialStore) CreateMedia(ctx context.Context, social *SocialService, req CreateMediaRequest) (MediaAsset, error) {
	_ = ctx
	_ = social
	return s.social.CreateMedia(req)
}

func (s *MemorySocialStore) FollowUser(ctx context.Context, social *SocialService, followerID, targetID string) error {
	_ = ctx
	_ = social
	return s.social.FollowUser(followerID, targetID)
}

func (s *MemorySocialStore) UnfollowUser(ctx context.Context, followerID, targetID string) error {
	_ = ctx
	return s.social.UnfollowUser(followerID, targetID)
}

func (s *MemorySocialStore) CreateConversation(ctx context.Context, social *SocialService, initiatorID string, req CreateConversationRequest) (Conversation, error) {
	_ = ctx
	_ = social
	return s.social.CreateConversation(initiatorID, req)
}

func (s *MemorySocialStore) AddMessage(ctx context.Context, social *SocialService, conversationID string, req CreateMessageRequest) (Conversation, error) {
	_ = ctx
	_ = social
	return s.social.AddMessage(conversationID, req)
}

func (s *MemorySocialStore) ListUsers(ctx context.Context, social *SocialService) ([]SocialUser, error) {
	_ = ctx
	_ = social
	return s.social.ListUsers(), nil
}

func (s *MemorySocialStore) GetUser(ctx context.Context, social *SocialService, id string) (*SocialUser, error) {
	_ = ctx
	_ = social
	return s.social.GetUser(id)
}

func (s *MemorySocialStore) UpdateUser(ctx context.Context, social *SocialService, id string, req UpdateUserRequest) (SocialUser, error) {
	_ = ctx
	_ = social
	return s.social.UpdateUser(id, req)
}

func (s *MemorySocialStore) ListMedia(ctx context.Context, social *SocialService, limit int) ([]MediaAsset, error) {
	_ = ctx
	_ = social
	return s.social.ListMedia(limit), nil
}

func (s *MemorySocialStore) Feed(ctx context.Context, social *SocialService, limit int) ([]SocialPost, error) {
	_ = ctx
	_ = social
	return s.social.Feed(limit), nil
}

func (s *MemorySocialStore) FeedMine(ctx context.Context, social *SocialService, limit int, currentUserID string) ([]SocialPost, error) {
	_ = ctx
	_ = social
	return s.social.FeedMine(limit, currentUserID), nil
}

func (s *MemorySocialStore) GetPost(ctx context.Context, social *SocialService, id string) (*SocialPost, error) {
	_ = ctx
	_ = social
	return s.social.GetPost(id)
}

func (s *MemorySocialStore) GetPostThread(ctx context.Context, social *SocialService, id string, limit int) (*PostThread, error) {
	_ = ctx
	_ = social
	return s.social.GetPostThread(id, limit)
}

func (s *MemorySocialStore) ListReplies(ctx context.Context, social *SocialService, id string, limit int) ([]SocialPost, error) {
	_ = ctx
	_ = social
	return s.social.ListReplies(id, limit)
}

func (s *MemorySocialStore) ListConversations(ctx context.Context, social *SocialService, limit int, currentUserID string) ([]Conversation, error) {
	_ = ctx
	_ = social
	return s.social.ListConversations(limit, currentUserID), nil
}

func (s *MemorySocialStore) GetConversation(ctx context.Context, social *SocialService, id string) (*Conversation, error) {
	_ = ctx
	_ = social
	return s.social.GetConversation(id)
}

func (s *MemorySocialStore) Bootstrap(ctx context.Context, social *SocialService, limit int, currentUserID string, mine bool) (BootstrapPayload, error) {
	_ = ctx
	_ = social
	if mine {
		return s.social.BootstrapMine(limit, currentUserID), nil
	}
	return s.social.Bootstrap(limit, currentUserID), nil
}

func (s *MemorySocialStore) VotePoll(ctx context.Context, social *SocialService, postID string, userID string, optionIndices []int) (SocialPost, error) {
	_ = ctx
	_ = social
	return s.social.VotePoll(postID, userID, optionIndices)
}

type PostgresUserStore struct {
	db *sql.DB
}

type PostgresSocialStore struct {
	db *sql.DB
}

func (s *PostgresUserStore) CreateWithAccount(ctx context.Context, req CreateUserRequest, account Account) (SocialUser, Account, error) {
	if strings.TrimSpace(req.Handle) == "" {
		return SocialUser{}, Account{}, errors.New("handle is required")
	}

	user := SocialUser{
		ID:           nextID("user"),
		Handle:       normalizeHandle(req.Handle),
		DisplayName:  valueOrDefault(strings.TrimSpace(req.DisplayName), strings.TrimPrefix(normalizeHandle(req.Handle), "@")),
		Bio:          strings.TrimSpace(req.Bio),
		Instance:     valueOrDefault(strings.TrimSpace(req.Instance), "vault.social"),
		Wallet:       strings.TrimSpace(req.Wallet),
		AvatarURL:    strings.TrimSpace(req.AvatarURL),
		Fields:       []UserField{},
		FeaturedTags: []string{},
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	fieldsRaw, _ := json.Marshal(user.Fields)
	featuredRaw, _ := json.Marshal(user.FeaturedTags)

	account.UserID = user.ID

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SocialUser{}, Account{}, err
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO social_users(
    id, handle, display_name, bio, instance, wallet, avatar_url,
    fields_json, featured_tags_json, is_bot, followers_count, following_count, created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9::jsonb,$10,$11,$12,$13)
`, user.ID, user.Handle, user.DisplayName, user.Bio, user.Instance, user.Wallet, user.AvatarURL, string(fieldsRaw), string(featuredRaw), user.IsBot, user.Followers, user.Following, user.CreatedAt); err != nil {
		_ = tx.Rollback()
		return SocialUser{}, Account{}, err
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO auth_accounts(id, username, email, password_hash, wallet, user_id, status, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (id) DO UPDATE SET
    username = EXCLUDED.username,
    email = EXCLUDED.email,
    password_hash = EXCLUDED.password_hash,
    wallet = EXCLUDED.wallet,
    user_id = EXCLUDED.user_id,
    status = EXCLUDED.status,
    updated_at = EXCLUDED.updated_at
`, account.ID, account.Username, nullableUniqueText(account.Email), account.PasswordHash, nullableUniqueText(account.Wallet), account.UserID, account.Status, account.CreatedAt, account.UpdatedAt); err != nil {
		_ = tx.Rollback()
		return SocialUser{}, Account{}, err
	}

	if err := tx.Commit(); err != nil {
		return SocialUser{}, Account{}, err
	}

	return user, account, nil
}

func (s *PostgresUserStore) Create(ctx context.Context, req CreateUserRequest) (SocialUser, error) {
	user, _, err := s.CreateWithAccount(ctx, req, Account{})
	if err != nil {
		return SocialUser{}, err
	}
	return user, nil
}

func (s *PostgresUserStore) GetByID(ctx context.Context, id string) (*SocialUser, error) {
	var user SocialUser
	var fieldsRaw []byte
	var featuredRaw []byte
	row := s.db.QueryRowContext(ctx, `
SELECT id, handle, display_name, bio, instance, wallet, avatar_url,
       fields_json, featured_tags_json, is_bot, followers_count, following_count, created_at
FROM social_users WHERE id = $1
`, id)
	if err := row.Scan(&user.ID, &user.Handle, &user.DisplayName, &user.Bio, &user.Instance, &user.Wallet, &user.AvatarURL, &fieldsRaw, &featuredRaw, &user.IsBot, &user.Followers, &user.Following, &user.CreatedAt); err != nil {
		return nil, err
	}
	_ = json.Unmarshal(fieldsRaw, &user.Fields)
	_ = json.Unmarshal(featuredRaw, &user.FeaturedTags)
	return &user, nil
}

func (s *PostgresUserStore) SaveAuthAccount(ctx context.Context, account Account) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO auth_accounts(id, username, email, password_hash, wallet, user_id, status, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (id) DO UPDATE SET
    username = EXCLUDED.username,
    email = EXCLUDED.email,
    password_hash = EXCLUDED.password_hash,
    wallet = EXCLUDED.wallet,
    user_id = EXCLUDED.user_id,
    status = EXCLUDED.status,
    updated_at = EXCLUDED.updated_at
`, account.ID, account.Username, nullableUniqueText(account.Email), account.PasswordHash, nullableUniqueText(account.Wallet), account.UserID, account.Status, account.CreatedAt, account.UpdatedAt)
	return err
}

func (s *PostgresUserStore) FindAuthAccountByWallet(ctx context.Context, wallet string) (Account, bool, error) {
	var account Account
	row := s.db.QueryRowContext(ctx, `
SELECT id, username, email, password_hash, wallet, user_id, status, created_at, updated_at
FROM auth_accounts WHERE LOWER(wallet) = LOWER($1)
LIMIT 1
`, strings.TrimSpace(wallet))
	if err := row.Scan(&account.ID, &account.Username, &account.Email, &account.PasswordHash, &account.Wallet, &account.UserID, &account.Status, &account.CreatedAt, &account.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Account{}, false, nil
		}
		return Account{}, false, err
	}
	return account, true, nil
}

func (s *PostgresUserStore) ListAuthAccounts(ctx context.Context) ([]Account, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, username, email, password_hash, wallet, user_id, status, created_at, updated_at
FROM auth_accounts
ORDER BY created_at ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := make([]Account, 0)
	for rows.Next() {
		var account Account
		if err := rows.Scan(&account.ID, &account.Username, &account.Email, &account.PasswordHash, &account.Wallet, &account.UserID, &account.Status, &account.CreatedAt, &account.UpdatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

func (s *PostgresSocialStore) CreatePost(ctx context.Context, social *SocialService, req CreatePostRequest) (SocialPost, error) {
	author, err := loadUserByID(ctx, s.db, req.AuthorID)
	if err != nil {
		return SocialPost{}, err
	}

	storageURI := strings.TrimSpace(req.StorageURI)
	if storageURI == "" {
		storageURI = buildLocalStorageURI(req)
	}

	post := SocialPost{
		ID:              nextID("post"),
		AuthorID:        author.ID,
		AuthorHandle:    author.Handle,
		AuthorName:      author.DisplayName,
		Instance:        author.Instance,
		Kind:            valueOrDefault(strings.TrimSpace(req.Kind), "post"),
		Content:         strings.TrimSpace(req.Content),
		Visibility:      valueOrDefault(strings.TrimSpace(req.Visibility), "public"),
		StorageURI:      storageURI,
		AttestationURI:  strings.TrimSpace(req.AttestationURI),
		ChainID:         "",
		TxHash:          "",
		ContractAddress: "",
		ExplorerURL:     "",
		Tags:            req.Tags,
		ParentPostID:    strings.TrimSpace(req.ParentPostID),
		RootPostID:      strings.TrimSpace(req.RootPostID),
		Type:            valueOrDefault(strings.TrimSpace(req.Type), "post"),
		Interaction:     valueOrDefault(strings.TrimSpace(req.Interaction), "anyone"),
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
	}

	if post.ParentPostID != "" {
		post.Kind = "reply"
		post.ReplyDepth = 1
		if post.RootPostID == "" {
			post.RootPostID = post.ParentPostID
		}
	}

	if len(req.PollOptions) >= 2 {
		expiresIn := req.PollExpiresIn
		if expiresIn <= 0 {
			expiresIn = 1440
		}
		poll := &Poll{
			Options:   make([]PollOption, len(req.PollOptions)),
			ExpiresAt: time.Now().UTC().Add(time.Duration(expiresIn) * time.Minute).Format(time.RFC3339),
			Multiple:  req.PollMultiple,
			Voters:    []string{},
		}
		for i, opt := range req.PollOptions {
			poll.Options[i] = PollOption{Label: strings.TrimSpace(opt), Votes: 0}
		}
		post.Poll = poll
	}

	tagsRaw, _ := json.Marshal(post.Tags)
	var pollRaw any
	if post.Poll != nil {
		encoded, _ := json.Marshal(post.Poll)
		pollRaw = string(encoded)
	}

	_, err = s.db.ExecContext(ctx, `
INSERT INTO social_posts(
  id, author_id, kind, content, visibility, storage_uri, attestation_uri,
  chain_id, tx_hash, contract_address, explorer_url,
  tags_json, parent_post_id, root_post_id, reply_depth, replies_count,
  boosts_count, likes_count, type, interaction, poll_json, created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::jsonb,$13,$14,$15,$16,$17,$18,$19,$20,$21::jsonb,$22)
","old_string":"INSERT INTO social_posts(
  id, author_id, kind, content, visibility, storage_uri, attestation_uri,
  tags_json, parent_post_id, root_post_id, reply_depth, replies_count,
  boosts_count, likes_count, type, interaction, poll_json, created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9,$10,$11,$12,$13,$14,$15,$16,$17::jsonb,$18)
","path":"/e:/MoleSociety/backend/database.go","replace_all":false}քների to=functions.StrReplace  天天买彩票_json code  天天中彩票大奖? սխալ JSON input maybe let's fix by proper JSON.{`, post.ID, post.AuthorID, post.Kind, post.Content, post.Visibility, post.StorageURI, post.AttestationURI, post.ChainID, post.TxHash, post.ContractAddress, post.ExplorerURL, string(tagsRaw), nullableString(post.ParentPostID), nullableString(post.RootPostID), post.ReplyDepth, post.Replies, post.Boosts, post.Likes, post.Type, post.Interaction, pollRaw, post.CreatedAt)
	if err != nil {
		return SocialPost{}, err
	}

	for _, mediaID := range req.MediaIDs {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO post_media_links(post_id, media_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, post.ID, mediaID); err == nil {
			if media, mediaErr := loadMediaByID(ctx, s.db, mediaID); mediaErr == nil {
				post.Media = append(post.Media, PostMedia{ID: media.ID, Name: media.Name, URL: media.URL, Kind: media.Kind, StorageURI: media.StorageURI, CID: media.CID})
			}
		}
	}

	return post, nil
}

func (s *PostgresSocialStore) CreateMedia(ctx context.Context, social *SocialService, req CreateMediaRequest) (MediaAsset, error) {
	if _, err := loadUserByID(ctx, s.db, req.OwnerID); err != nil {
		return MediaAsset{}, err
	}

	asset := MediaAsset{
		ID:         nextID("media"),
		OwnerID:    req.OwnerID,
		Name:       strings.TrimSpace(req.Name),
		Kind:       valueOrDefault(strings.TrimSpace(req.Kind), "image"),
		URL:        strings.TrimSpace(req.URL),
		StorageURI: strings.TrimSpace(req.StorageURI),
		CID:        strings.TrimSpace(req.CID),
		SizeBytes:  req.SizeBytes,
		Status:     valueOrDefault(strings.TrimSpace(req.Status), "uploaded"),
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	_, err := s.db.ExecContext(ctx, `
INSERT INTO media_assets(id, owner_id, name, kind, url, storage_uri, cid, size_bytes, status, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
`, asset.ID, asset.OwnerID, asset.Name, asset.Kind, asset.URL, asset.StorageURI, asset.CID, asset.SizeBytes, asset.Status, asset.CreatedAt)
	if err != nil {
		return MediaAsset{}, err
	}
	return asset, nil
}

func (s *PostgresSocialStore) FollowUser(ctx context.Context, social *SocialService, followerID, targetID string) error {
	if _, err := loadUserByID(ctx, s.db, followerID); err != nil {
		return err
	}
	if _, err := loadUserByID(ctx, s.db, targetID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO user_follows(follower_id, followee_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, followerID, targetID)
	return err
}

func (s *PostgresSocialStore) UnfollowUser(ctx context.Context, followerID, targetID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM user_follows WHERE follower_id = $1 AND followee_id = $2`, followerID, targetID)
	return err
}

func (s *PostgresSocialStore) CreateConversation(ctx context.Context, social *SocialService, initiatorID string, req CreateConversationRequest) (Conversation, error) {
	if strings.TrimSpace(initiatorID) == "" {
		return Conversation{}, errors.New("initiator is required")
	}
	if _, err := loadUserByID(ctx, s.db, initiatorID); err != nil {
		return Conversation{}, err
	}
	participantSet := map[string]bool{initiatorID: true}
	for _, participantID := range req.ParticipantIDs {
		trimmed := strings.TrimSpace(participantID)
		if trimmed == "" {
			continue
		}
		if _, err := loadUserByID(ctx, s.db, trimmed); err != nil {
			return Conversation{}, err
		}
		participantSet[trimmed] = true
	}
	participantIDs := make([]string, 0, len(participantSet))
	for participantID := range participantSet {
		participantIDs = append(participantIDs, participantID)
	}
	sort.Strings(participantIDs)
	if len(participantIDs) != 2 {
		return Conversation{}, errors.New("conversation requires exactly two participants")
	}
	conversation := Conversation{
		ID:             nextID("conv"),
		Title:          valueOrDefault(strings.TrimSpace(req.Title), "New Conversation"),
		ParticipantIDs: participantIDs,
		InitiatorID:    initiatorID,
		Encrypted:      req.Encrypted,
		Messages:       []ChatMessage{},
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Conversation{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO conversations(id, title, initiator_id, encrypted, updated_at) VALUES ($1,$2,$3,$4,$5)`, conversation.ID, conversation.Title, conversation.InitiatorID, conversation.Encrypted, conversation.UpdatedAt); err != nil {
		_ = tx.Rollback()
		return Conversation{}, err
	}
	for _, participantID := range participantIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO conversation_participants(conversation_id, user_id) VALUES ($1,$2)`, conversation.ID, participantID); err != nil {
			_ = tx.Rollback()
			return Conversation{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Conversation{}, err
	}
	return conversation, nil
}

func (s *PostgresSocialStore) AddMessage(ctx context.Context, social *SocialService, conversationID string, req CreateMessageRequest) (Conversation, error) {
	conversation, err := loadConversationByID(ctx, s.db, conversationID)
	if err != nil {
		return Conversation{}, errors.New("conversation not found: " + conversationID)
	}
	sender, err := loadUserByID(ctx, s.db, req.SenderID)
	if err != nil {
		return Conversation{}, err
	}
	isParticipant := false
	for _, participantID := range conversation.ParticipantIDs {
		if participantID == sender.ID {
			isParticipant = true
			break
		}
	}
	if !isParticipant {
		return Conversation{}, errors.New("sender is not a participant in this conversation")
	}
	if len(conversation.ParticipantIDs) != 2 {
		return Conversation{}, errors.New("conversation must contain exactly two participants")
	}
	peerID := conversation.ParticipantIDs[0]
	if peerID == sender.ID {
		peerID = conversation.ParticipantIDs[1]
	}
	mutualFollow, err := isMutualFollow(ctx, s.db, sender.ID, peerID)
	if err != nil {
		return Conversation{}, err
	}
	if !mutualFollow {
		if conversation.InitiatorID == "" {
			conversation.InitiatorID = sender.ID
			if _, err := s.db.ExecContext(ctx, `UPDATE conversations SET initiator_id = $1 WHERE id = $2`, conversation.InitiatorID, conversation.ID); err != nil {
				return Conversation{}, err
			}
		}
		if sender.ID != conversation.InitiatorID {
			return Conversation{}, errors.New("the other user has not followed you back yet")
		}
		if len(conversation.Messages) >= 1 {
			return Conversation{}, errors.New("awaiting follow-back: only one message is allowed")
		}
	}
	message := ChatMessage{
		ID:             nextID("msg"),
		ConversationID: conversationID,
		SenderID:       sender.ID,
		SenderHandle:   sender.Handle,
		Body:           strings.TrimSpace(req.Body),
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Conversation{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO chat_messages(id, conversation_id, sender_id, body, created_at) VALUES ($1,$2,$3,$4,$5)`, message.ID, message.ConversationID, message.SenderID, message.Body, message.CreatedAt); err != nil {
		_ = tx.Rollback()
		return Conversation{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE conversations SET updated_at = $1, initiator_id = $2 WHERE id = $3`, message.CreatedAt, nullableString(conversation.InitiatorID), conversation.ID); err != nil {
		_ = tx.Rollback()
		return Conversation{}, err
	}
	if err := tx.Commit(); err != nil {
		return Conversation{}, err
	}
	updated, err := loadConversationByID(ctx, s.db, conversationID)
	if err != nil {
		return Conversation{}, err
	}
	return *updated, nil
}

func (s *PostgresSocialStore) ListUsers(ctx context.Context, social *SocialService) ([]SocialUser, error) {
	_ = social
	return loadUsers(ctx, s.db)
}

func (s *PostgresSocialStore) GetUser(ctx context.Context, social *SocialService, id string) (*SocialUser, error) {
	_ = social
	return loadUserByID(ctx, s.db, id)
}

func (s *PostgresSocialStore) UpdateUser(ctx context.Context, social *SocialService, id string, req UpdateUserRequest) (SocialUser, error) {
	_ = social
	fieldsRaw := []byte("[]")
	featuredRaw := []byte("[]")
	if req.Fields != nil {
		fieldsRaw, _ = json.Marshal(*req.Fields)
	}
	if req.FeaturedTags != nil {
		featuredRaw, _ = json.Marshal(*req.FeaturedTags)
	}

	query := `
UPDATE social_users
SET display_name = COALESCE($2, display_name),
    bio = COALESCE($3, bio),
    avatar_url = COALESCE($4, avatar_url),
    fields_json = CASE WHEN $5::jsonb IS NULL THEN fields_json ELSE $5::jsonb END,
    featured_tags_json = CASE WHEN $6::jsonb IS NULL THEN featured_tags_json ELSE $6::jsonb END,
    is_bot = COALESCE($7, is_bot)
WHERE id = $1
`
	var fieldsArg any
	var featuredArg any
	if req.Fields != nil {
		fieldsArg = string(fieldsRaw)
	}
	if req.FeaturedTags != nil {
		featuredArg = string(featuredRaw)
	}
	if _, err := s.db.ExecContext(ctx, query, id, nullableStringPtr(req.DisplayName), nullableStringPtr(req.Bio), nullableStringPtr(req.AvatarURL), fieldsArg, featuredArg, req.IsBot); err != nil {
		return SocialUser{}, err
	}
	user, err := loadUserByID(ctx, s.db, id)
	if err != nil {
		return SocialUser{}, err
	}
	return *user, nil
}

func (s *PostgresSocialStore) ListMedia(ctx context.Context, social *SocialService, limit int) ([]MediaAsset, error) {
	_ = social
	return loadMediaList(ctx, s.db, limit)
}

func (s *PostgresSocialStore) Feed(ctx context.Context, social *SocialService, limit int) ([]SocialPost, error) {
	_ = social
	return loadFeed(ctx, s.db, limit, "")
}

func (s *PostgresSocialStore) FeedMine(ctx context.Context, social *SocialService, limit int, currentUserID string) ([]SocialPost, error) {
	_ = social
	return loadFeed(ctx, s.db, limit, currentUserID)
}

func (s *PostgresSocialStore) GetPost(ctx context.Context, social *SocialService, id string) (*SocialPost, error) {
	_ = social
	return loadPostByID(ctx, s.db, id)
}

func (s *PostgresSocialStore) GetPostThread(ctx context.Context, social *SocialService, id string, limit int) (*PostThread, error) {
	_ = social
	post, err := loadPostByID(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	ancestors, err := loadAncestors(ctx, s.db, post)
	if err != nil {
		return nil, err
	}
	replies, err := loadReplies(ctx, s.db, id, limit)
	if err != nil {
		return nil, err
	}
	return &PostThread{Post: *post, Ancestors: ancestors, Replies: replies}, nil
}

func (s *PostgresSocialStore) ListReplies(ctx context.Context, social *SocialService, id string, limit int) ([]SocialPost, error) {
	_ = social
	return loadReplies(ctx, s.db, id, limit)
}

func (s *PostgresSocialStore) ListConversations(ctx context.Context, social *SocialService, limit int, currentUserID string) ([]Conversation, error) {
	_ = social
	return loadConversations(ctx, s.db, limit, currentUserID)
}

func (s *PostgresSocialStore) GetConversation(ctx context.Context, social *SocialService, id string) (*Conversation, error) {
	_ = social
	return loadConversationByID(ctx, s.db, id)
}

func (s *PostgresSocialStore) Bootstrap(ctx context.Context, social *SocialService, limit int, currentUserID string, mine bool) (BootstrapPayload, error) {
	users, err := loadUsers(ctx, s.db)
	if err != nil {
		return BootstrapPayload{}, err
	}
	media, err := loadMediaList(ctx, s.db, limit)
	if err != nil {
		return BootstrapPayload{}, err
	}
	conversations, err := loadConversations(ctx, s.db, limit, currentUserID)
	if err != nil {
		return BootstrapPayload{}, err
	}
	feed, err := loadFeed(ctx, s.db, limit, func() string {
		if mine {
			return currentUserID
		}
		return ""
	}())
	if err != nil {
		return BootstrapPayload{}, err
	}
	instances := append([]FederationInstance(nil), social.ListInstances()...)
	stats := SocialStats{Users: len(users), Posts: len(feed), MediaAssets: len(media), Conversations: len(conversations)}
	payload := BootstrapPayload{Stats: stats, Feed: feed, Users: users, Media: media, Conversations: conversations, Instances: instances}
	if strings.TrimSpace(currentUserID) != "" {
		currentUser, err := loadUserByID(ctx, s.db, currentUserID)
		if err == nil {
			payload.CurrentUser = currentUser
		}
	} else if len(users) > 0 {
		payload.CurrentUser = &users[0]
	}
	return payload, nil
}

func (s *PostgresSocialStore) VotePoll(ctx context.Context, social *SocialService, postID string, userID string, optionIndices []int) (SocialPost, error) {
	_ = social
	post, err := loadPostByID(ctx, s.db, postID)
	if err != nil {
		return SocialPost{}, errors.New("post not found")
	}
	if post.Poll == nil {
		return SocialPost{}, errors.New("post has no poll")
	}

	expiresAt, err := time.Parse(time.RFC3339, post.Poll.ExpiresAt)
	if err == nil && time.Now().UTC().After(expiresAt) {
		return SocialPost{}, errors.New("poll has expired")
	}

	for _, voterID := range post.Poll.Voters {
		if voterID == userID {
			return SocialPost{}, errors.New("already voted")
		}
	}

	if len(optionIndices) == 0 {
		return SocialPost{}, errors.New("no options selected")
	}
	if !post.Poll.Multiple && len(optionIndices) > 1 {
		return SocialPost{}, errors.New("single choice only")
	}
	for _, optIdx := range optionIndices {
		if optIdx < 0 || optIdx >= len(post.Poll.Options) {
			return SocialPost{}, errors.New("invalid option index")
		}
	}

	for _, optIdx := range optionIndices {
		post.Poll.Options[optIdx].Votes++
	}
	post.Poll.Voters = append(post.Poll.Voters, userID)

	pollRaw, _ := json.Marshal(post.Poll)
	if _, err := s.db.ExecContext(ctx, `UPDATE social_posts SET poll_json = $1::jsonb WHERE id = $2`, string(pollRaw), postID); err != nil {
		return SocialPost{}, err
	}

	updated, err := loadPostByID(ctx, s.db, postID)
	if err != nil {
		return SocialPost{}, err
	}
	return *updated, nil
}

func loadMediaByID(ctx context.Context, db *sql.DB, id string) (MediaAsset, error) {
	var asset MediaAsset
	row := db.QueryRowContext(ctx, `
SELECT id, owner_id, name, kind, url, storage_uri, cid, size_bytes, status, created_at
FROM media_assets WHERE id = $1
`, id)
	err := row.Scan(&asset.ID, &asset.OwnerID, &asset.Name, &asset.Kind, &asset.URL, &asset.StorageURI, &asset.CID, &asset.SizeBytes, &asset.Status, &asset.CreatedAt)
	return asset, err
}

func loadUserByID(ctx context.Context, db *sql.DB, id string) (*SocialUser, error) {
	var user SocialUser
	var fieldsRaw []byte
	var featuredRaw []byte
	row := db.QueryRowContext(ctx, `
SELECT id, handle, display_name, bio, instance, wallet, avatar_url,
       fields_json, featured_tags_json, is_bot, followers_count, following_count, created_at
FROM social_users WHERE id = $1
`, id)
	if err := row.Scan(&user.ID, &user.Handle, &user.DisplayName, &user.Bio, &user.Instance, &user.Wallet, &user.AvatarURL, &fieldsRaw, &featuredRaw, &user.IsBot, &user.Followers, &user.Following, &user.CreatedAt); err != nil {
		return nil, err
	}
	_ = json.Unmarshal(fieldsRaw, &user.Fields)
	_ = json.Unmarshal(featuredRaw, &user.FeaturedTags)
	return &user, nil
}

func loadUsers(ctx context.Context, db *sql.DB) ([]SocialUser, error) {
	rows, err := db.QueryContext(ctx, `
SELECT id, handle, display_name, bio, instance, wallet, avatar_url,
       fields_json, featured_tags_json, is_bot, followers_count, following_count, created_at
FROM social_users
ORDER BY created_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]SocialUser, 0)
	for rows.Next() {
		var user SocialUser
		var fieldsRaw []byte
		var featuredRaw []byte
		if err := rows.Scan(&user.ID, &user.Handle, &user.DisplayName, &user.Bio, &user.Instance, &user.Wallet, &user.AvatarURL, &fieldsRaw, &featuredRaw, &user.IsBot, &user.Followers, &user.Following, &user.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(fieldsRaw, &user.Fields)
		_ = json.Unmarshal(featuredRaw, &user.FeaturedTags)
		users = append(users, user)
	}
	return users, rows.Err()
}

func loadMediaList(ctx context.Context, db *sql.DB, limit int) ([]MediaAsset, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := db.QueryContext(ctx, `
SELECT id, owner_id, name, kind, url, storage_uri, cid, size_bytes, status, created_at
FROM media_assets
ORDER BY created_at DESC
LIMIT $1
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assets := make([]MediaAsset, 0)
	for rows.Next() {
		var asset MediaAsset
		if err := rows.Scan(&asset.ID, &asset.OwnerID, &asset.Name, &asset.Kind, &asset.URL, &asset.StorageURI, &asset.CID, &asset.SizeBytes, &asset.Status, &asset.CreatedAt); err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	return assets, rows.Err()
}

func loadPostByID(ctx context.Context, db *sql.DB, id string) (*SocialPost, error) {
	row := db.QueryRowContext(ctx, `
SELECT p.id, p.author_id, u.handle, u.display_name, u.instance, p.kind, p.content, p.visibility,
       p.storage_uri, p.attestation_uri, p.chain_id, p.tx_hash, p.contract_address, p.explorer_url,
       p.tags_json, p.parent_post_id, p.root_post_id, p.reply_depth,
       p.replies_count, p.boosts_count, p.likes_count, p.type, p.interaction, p.poll_json, p.created_at
FROM social_posts p
JOIN social_users u ON u.id = p.author_id
WHERE p.id = $1
`, id)
	post, err := scanPostRow(row)
	if err != nil {
		return nil, err
	}
	media, err := loadPostMedia(ctx, db, post.ID)
	if err != nil {
		return nil, err
	}
	post.Media = media
	return post, nil
}

func loadFeed(ctx context.Context, db *sql.DB, limit int, authorID string) ([]SocialPost, error) {
	if limit <= 0 {
		limit = 20
	}
	base := `
SELECT p.id, p.author_id, u.handle, u.display_name, u.instance, p.kind, p.content, p.visibility,
       p.storage_uri, p.attestation_uri, p.chain_id, p.tx_hash, p.contract_address, p.explorer_url,
       p.tags_json, p.parent_post_id, p.root_post_id, p.reply_depth,
       p.replies_count, p.boosts_count, p.likes_count, p.type, p.interaction, p.poll_json, p.created_at
FROM social_posts p
JOIN social_users u ON u.id = p.author_id
WHERE p.parent_post_id IS NULL`
	var rows *sql.Rows
	var err error
	if strings.TrimSpace(authorID) != "" {
		rows, err = db.QueryContext(ctx, base+` AND p.author_id = $1 ORDER BY p.created_at DESC LIMIT $2`, authorID, limit)
	} else {
		rows, err = db.QueryContext(ctx, base+` ORDER BY p.created_at DESC LIMIT $1`, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPosts(ctx, db, rows)
}

func loadReplies(ctx context.Context, db *sql.DB, postID string, limit int) ([]SocialPost, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := db.QueryContext(ctx, `
SELECT p.id, p.author_id, u.handle, u.display_name, u.instance, p.kind, p.content, p.visibility,
       p.storage_uri, p.attestation_uri, p.chain_id, p.tx_hash, p.contract_address, p.explorer_url,
       p.tags_json, p.parent_post_id, p.root_post_id, p.reply_depth,
       p.replies_count, p.boosts_count, p.likes_count, p.type, p.interaction, p.poll_json, p.created_at
FROM social_posts p
JOIN social_users u ON u.id = p.author_id
WHERE p.parent_post_id = $1 OR p.root_post_id = $1
ORDER BY p.created_at ASC
LIMIT $2
`, postID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPosts(ctx, db, rows)
}

func loadAncestors(ctx context.Context, db *sql.DB, post *SocialPost) ([]SocialPost, error) {
	if post == nil || strings.TrimSpace(post.ParentPostID) == "" {
		return nil, nil
	}
	ancestors := make([]SocialPost, 0)
	current := strings.TrimSpace(post.ParentPostID)
	for current != "" {
		parent, err := loadPostByID(ctx, db, current)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				break
			}
			return nil, err
		}
		ancestors = append([]SocialPost{*parent}, ancestors...)
		current = strings.TrimSpace(parent.ParentPostID)
	}
	return ancestors, nil
}

func loadConversations(ctx context.Context, db *sql.DB, limit int, currentUserID string) ([]Conversation, error) {
	if limit <= 0 {
		limit = 20
	}
	currentUserID = strings.TrimSpace(currentUserID)

	query := `
SELECT c.id, c.title, c.initiator_id, c.encrypted, c.updated_at
FROM conversations c`
	args := []any{}
	if currentUserID != "" {
		query += `
JOIN conversation_participants cp ON cp.conversation_id = c.id
WHERE cp.user_id = $1`
		args = append(args, currentUserID)
	}
	query += `
ORDER BY c.updated_at DESC
LIMIT $` + strconv.Itoa(len(args)+1)
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Conversation, 0)
	for rows.Next() {
		var item Conversation
		var initiator sql.NullString
		if err := rows.Scan(&item.ID, &item.Title, &initiator, &item.Encrypted, &item.UpdatedAt); err != nil {
			return nil, err
		}
		if initiator.Valid {
			item.InitiatorID = initiator.String
		}
		participants, err := loadConversationParticipants(ctx, db, item.ID)
		if err != nil {
			return nil, err
		}
		item.ParticipantIDs = participants
		messages, err := loadMessages(ctx, db, item.ID)
		if err != nil {
			return nil, err
		}
		item.Messages = messages
		items = append(items, item)
	}
	return items, rows.Err()
}

func loadConversationByID(ctx context.Context, db *sql.DB, id string) (*Conversation, error) {
	row := db.QueryRowContext(ctx, `
SELECT id, title, initiator_id, encrypted, updated_at
FROM conversations WHERE id = $1
`, id)
	var item Conversation
	var initiator sql.NullString
	if err := row.Scan(&item.ID, &item.Title, &initiator, &item.Encrypted, &item.UpdatedAt); err != nil {
		return nil, err
	}
	if initiator.Valid {
		item.InitiatorID = initiator.String
	}
	participants, err := loadConversationParticipants(ctx, db, item.ID)
	if err != nil {
		return nil, err
	}
	item.ParticipantIDs = participants
	messages, err := loadMessages(ctx, db, item.ID)
	if err != nil {
		return nil, err
	}
	item.Messages = messages
	return &item, nil
}

func loadConversationParticipants(ctx context.Context, db *sql.DB, conversationID string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
SELECT user_id FROM conversation_participants WHERE conversation_id = $1 ORDER BY user_id ASC
`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func loadMessages(ctx context.Context, db *sql.DB, conversationID string) ([]ChatMessage, error) {
	rows, err := db.QueryContext(ctx, `
SELECT m.id, m.conversation_id, m.sender_id, u.handle, m.body, m.created_at
FROM chat_messages m
JOIN social_users u ON u.id = m.sender_id
WHERE m.conversation_id = $1
ORDER BY m.created_at ASC
`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ChatMessage, 0)
	for rows.Next() {
		var item ChatMessage
		if err := rows.Scan(&item.ID, &item.ConversationID, &item.SenderID, &item.SenderHandle, &item.Body, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func isMutualFollow(ctx context.Context, db *sql.DB, a string, b string) (bool, error) {
	var count int
	row := db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM user_follows WHERE (follower_id = $1 AND followee_id = $2) OR (follower_id = $2 AND followee_id = $1)
`, a, b)
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count == 2, nil
}

func scanPosts(ctx context.Context, db *sql.DB, rows *sql.Rows) ([]SocialPost, error) {
	items := make([]SocialPost, 0)
	for rows.Next() {
		item, err := scanPostRow(rows)
		if err != nil {
			return nil, err
		}
		media, err := loadPostMedia(ctx, db, item.ID)
		if err != nil {
			return nil, err
		}
		item.Media = media
		items = append(items, *item)
	}
	return items, rows.Err()
}

func loadPostMedia(ctx context.Context, db *sql.DB, postID string) ([]PostMedia, error) {
	rows, err := db.QueryContext(ctx, `
SELECT m.id, m.name, m.url, m.kind, m.storage_uri, m.cid
FROM post_media_links pml
JOIN media_assets m ON m.id = pml.media_id
WHERE pml.post_id = $1
ORDER BY m.created_at ASC
`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]PostMedia, 0)
	for rows.Next() {
		var item PostMedia
		if err := rows.Scan(&item.ID, &item.Name, &item.URL, &item.Kind, &item.StorageURI, &item.CID); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanPostRow(scanner interface{ Scan(dest ...any) error }) (*SocialPost, error) {
	var item SocialPost
	var tagsRaw []byte
	var parent sql.NullString
	var root sql.NullString
	var pollRaw []byte
	if err := scanner.Scan(&item.ID, &item.AuthorID, &item.AuthorHandle, &item.AuthorName, &item.Instance, &item.Kind, &item.Content, &item.Visibility, &item.StorageURI, &item.AttestationURI, &item.ChainID, &item.TxHash, &item.ContractAddress, &item.ExplorerURL, &tagsRaw, &parent, &root, &item.ReplyDepth, &item.Replies, &item.Boosts, &item.Likes, &item.Type, &item.Interaction, &pollRaw, &item.CreatedAt); err != nil {
		return nil, err
	}
	if parent.Valid {
		item.ParentPostID = parent.String
	}
	if root.Valid {
		item.RootPostID = root.String
	}
	_ = json.Unmarshal(tagsRaw, &item.Tags)
	if len(pollRaw) > 0 && string(pollRaw) != "null" {
		var poll Poll
		if err := json.Unmarshal(pollRaw, &poll); err == nil {
			item.Poll = &poll
		}
	}
	return &item, nil
}

func nullableString(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func nullableStringPtr(value *string) any {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return trimmed
}

func nullableUniqueText(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func (a *App) databaseHealth() map[string]any {
	if a.db == nil {
		return map[string]any{
			"enabled": false,
		}
	}

	payload := map[string]any{
		"enabled":            a.db.Available,
		"driver":             a.db.Driver,
		"migrationsDir":      a.db.MigrationsDir,
		"maxOpenConns":       a.db.MaxOpenConns,
		"maxIdleConns":       a.db.MaxIdleConns,
		"connMaxLifetimeSec": int(a.db.ConnMaxLifetime.Seconds()),
	}

	if !a.db.Available {
		payload["mode"] = "disabled"
		return payload
	}

	stats := a.db.SQL.Stats()
	payload["mode"] = "connected"
	payload["openConnections"] = stats.OpenConnections
	payload["inUse"] = stats.InUse
	payload["idle"] = stats.Idle
	return payload
}

func (a *App) migrationsStatus() map[string]any {
	if a.db == nil {
		return map[string]any{"count": 0, "files": []string{}, "applied": []string{}}
	}

	files, err := a.db.MigrationFiles()
	if err != nil {
		return map[string]any{"count": 0, "error": err.Error(), "files": []string{}, "applied": []string{}}
	}

	baseNames := make([]string, 0, len(files))
	for _, file := range files {
		baseNames = append(baseNames, filepath.Base(file))
	}

	appliedVersions := make([]string, 0)
	if a.db.Available {
		applied, err := a.db.AppliedMigrations(a.ctx)
		if err != nil {
			return map[string]any{"count": len(baseNames), "files": baseNames, "error": err.Error(), "applied": []string{}}
		}
		for _, item := range applied {
			appliedVersions = append(appliedVersions, item.Version)
		}
	}

	return map[string]any{
		"count":   len(baseNames),
		"files":   baseNames,
		"applied": appliedVersions,
	}
}

func mustFormatDatabaseMode(db *Database) string {
	if db == nil || !db.Available {
		return "memory+redis"
	}
	return fmt.Sprintf("postgres(%s)+redis", db.Driver)
}

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

const (
	authSessionCookieName = "molesociety_session"
	authChallengeTTL      = 5 * time.Minute
	authSessionTTL        = 7 * 24 * time.Hour
)

var (
	errAuthSessionMissing = errors.New("authentication required")
	errAuthSessionInvalid = errors.New("invalid session")
)

type AuthError struct {
	Code    string `json:"code"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (e *AuthError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func newAuthError(code, kind, message string) *AuthError {
	return &AuthError{Code: code, Type: kind, Message: message}
}

func isAuthErrorCode(err error, code string) bool {
	var authErr *AuthError
	if errors.As(err, &authErr) {
		return authErr.Code == code
	}
	return false
}

func writeAuthError(w http.ResponseWriter, status int, err error) {
	var authErr *AuthError
	if errors.As(err, &authErr) {
		writeJSON(w, status, map[string]any{
			"ok":    false,
			"error": authErr.Message,
			"code":  authErr.Code,
			"type":  authErr.Type,
		})
		return
	}

	message := "authentication failed"
	if err != nil {
		message = err.Error()
	}
	writeJSON(w, status, map[string]any{
		"ok":    false,
		"error": message,
		"code":  "AUTH_UNKNOWN",
		"type":  "unknown",
	})
}

type AuthChallenge struct {
	Nonce     string `json:"nonce"`
	Address   string `json:"address"`
	Message   string `json:"message"`
	ChainID   int64  `json:"chainId"`
	IssuedAt  string `json:"issuedAt"`
	ExpiresAt string `json:"expiresAt"`
}

type AuthSessionRecord struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	Address   string `json:"address"`
	CreatedAt string `json:"createdAt"`
	ExpiresAt string `json:"expiresAt"`
}

type AuthChallengeRequest struct {
	Address string `json:"address"`
	ChainID int64  `json:"chainId"`
}

type VerifyWalletRequest struct {
	Address   string `json:"address"`
	Nonce     string `json:"nonce"`
	Signature string `json:"signature"`
}

type PasswordLoginRequest struct {
	Identifier string `json:"identifier"` // username or email
	Password   string `json:"password"`
}

type RegisterRequest struct {
	Username      string `json:"username"`
	Email         string `json:"email"`
	Password      string `json:"password"`
	WalletAddress string `json:"walletAddress"`
	AutoWallet    bool   `json:"autoWallet"`
	ChainID       int64  `json:"chainId"`
	Signature     string `json:"signature"`
	Nonce         string `json:"nonce"`
}

type BindChallengeRequest struct {
	WalletAddress string `json:"walletAddress"`
	ChainID       int64  `json:"chainId"`
}

type Account struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	PasswordHash string `json:"passwordHash"`
	Wallet       string `json:"wallet"`
	UserID       string `json:"userId"` // linked SocialUser ID
	Status       string `json:"status"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type AuthSessionResponse struct {
	ID           string      `json:"id"`
	Handle       string      `json:"handle"`
	DisplayName  string      `json:"displayName"`
	Instance     string      `json:"instance"`
	Bio          string      `json:"bio"`
	AvatarURL    string      `json:"avatarUrl"`
	Wallet       string      `json:"wallet"`
	Fields       []UserField `json:"fields"`
	FeaturedTags []string    `json:"featuredTags"`
	IsBot        bool        `json:"isBot"`
}

type AuthService struct {
	ctx        context.Context
	rdb        *redis.Client
	social     *SocialService
	store      UserStore
	userLookup func(string) (*SocialUser, error)
	mu         sync.Mutex
	challenges map[string]AuthChallenge
	sessions   map[string]AuthSessionRecord
	accounts   map[string]Account // keyed by account ID
}

func NewAuthService(ctx context.Context, rdb *redis.Client, social *SocialService) *AuthService {
	s := &AuthService{
		ctx:        ctx,
		rdb:        rdb,
		social:     social,
		challenges: map[string]AuthChallenge{},
		sessions:   map[string]AuthSessionRecord{},
		accounts:   map[string]Account{},
	}
	return s
}

func (s *AuthService) SetStore(store UserStore) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store = store
	s.loadAccountsFromRedis()
}

func (s *AuthService) SetUserLookup(lookup func(string) (*SocialUser, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.userLookup = lookup
}

func (s *AuthService) getUserByIDNoLock(id string) (*SocialUser, error) {
	if s.userLookup != nil {
		return s.userLookup(id)
	}
	return s.social.GetUser(id)
}

func (s *AuthService) loadAccountsFromRedis() {
	if s.store != nil {
		accounts, err := s.store.ListAuthAccounts(s.ctx)
		if err == nil && len(accounts) > 0 {
			for _, acc := range accounts {
				s.accounts[acc.ID] = acc
			}
			log.Printf("loaded %d accounts from store", len(accounts))
			return
		}
	}

	if s.rdb == nil {
		return
	}
	raw, err := s.rdb.Get(s.ctx, "auth:accounts:snapshot").Result()
	if err != nil {
		return
	}
	var accounts []Account
	if err := json.Unmarshal([]byte(raw), &accounts); err != nil {
		return
	}
	for _, acc := range accounts {
		s.accounts[acc.ID] = acc
	}
	log.Printf("loaded %d accounts from redis", len(accounts))
}

func (s *AuthService) persistAccountsLocked() {
	if s.store != nil {
		for _, acc := range s.accounts {
			if err := s.store.SaveAuthAccount(s.ctx, acc); err != nil {
				log.Printf("persist account to store failed: %v", err)
			}
		}
	}

	if s.rdb == nil {
		return
	}
	accounts := make([]Account, 0, len(s.accounts))
	for _, acc := range s.accounts {
		accounts = append(accounts, acc)
	}
	raw, err := json.Marshal(accounts)
	if err != nil {
		return
	}
	_ = s.rdb.Set(s.ctx, "auth:accounts:snapshot", raw, 0).Err()
}

func (a *App) registerAuthRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/auth/challenge", a.authChallengeHandler).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/api/v1/auth/verify", a.authVerifyHandler).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/api/v1/auth/me", a.authMeHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/api/v1/auth/logout", a.authLogoutHandler).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/api/v1/auth/password-login", a.authPasswordLoginHandler).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/api/v1/auth/register", a.authRegisterHandler).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/api/v1/auth/bind-challenge", a.authBindChallengeHandler).Methods(http.MethodPost, http.MethodOptions)
}

func (a *App) authChallengeHandler(w http.ResponseWriter, r *http.Request) {
	var req AuthChallengeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAuthError(w, http.StatusBadRequest, newAuthError("AUTH_INVALID_JSON", "validation", "invalid JSON"))
		return
	}

	chainID := req.ChainID
	if chainID == 0 {
		if a.chainID != nil {
			chainID = a.chainID.Int64()
		} else {
			chainID = 1
		}
	}

	challenge, err := a.auth.CreateChallenge(req.Address, chainID, r)
	if err != nil {
		writeAuthError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": challenge})
}

func (a *App) authVerifyHandler(w http.ResponseWriter, r *http.Request) {
	var req VerifyWalletRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAuthError(w, http.StatusBadRequest, newAuthError("AUTH_INVALID_JSON", "validation", "invalid JSON"))
		return
	}

	user, session, err := a.auth.VerifyWalletLogin(req.Address, req.Nonce, req.Signature)
	if err != nil {
		status := http.StatusUnauthorized
		if isAuthErrorCode(err, "AUTH_WALLET_NOT_BOUND") {
			status = http.StatusNotFound
		}
		writeAuthError(w, status, err)
		return
	}

	setSessionCookie(w, session.ID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": authSessionResponseFromUser(user)})
}

func (a *App) authMeHandler(w http.ResponseWriter, r *http.Request) {
	user, err := a.optionalAuthenticatedUser(r)
	if err != nil {
		clearSessionCookie(w)
		writeAuthError(w, http.StatusUnauthorized, err)
		return
	}
	if user == nil {
		writeAuthError(w, http.StatusUnauthorized, newAuthError("AUTH_SESSION_REQUIRED", "session", errAuthSessionMissing.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": authSessionResponseFromUser(*user)})
}

func (a *App) authLogoutHandler(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(authSessionCookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
		a.auth.DeleteSession(strings.TrimSpace(cookie.Value))
	}

	clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": map[string]any{"loggedOut": true}})
}

// --- Password Login Handler ---
func (a *App) authPasswordLoginHandler(w http.ResponseWriter, r *http.Request) {
	var req PasswordLoginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAuthError(w, http.StatusBadRequest, newAuthError("AUTH_INVALID_JSON", "validation", "invalid JSON"))
		return
	}

	identifier := strings.TrimSpace(req.Identifier)
	password := strings.TrimSpace(req.Password)
	if identifier == "" || password == "" {
		writeAuthError(w, http.StatusBadRequest, newAuthError("AUTH_MISSING_CREDENTIALS", "validation", "identifier and password are required"))
		return
	}

	user, session, err := a.auth.PasswordLogin(identifier, password)
	if err != nil {
		log.Printf("password-login failed for %q: %v", identifier, err)
		status := http.StatusUnauthorized
		if isAuthErrorCode(err, "AUTH_ACCOUNT_NOT_FOUND") {
			status = http.StatusNotFound
		}
		writeAuthError(w, status, err)
		return
	}

	setSessionCookie(w, session.ID)
	log.Printf("password-login success: user=%s handle=%s", user.ID, user.Handle)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": authSessionResponseFromUser(user)})
}

// --- Register + Bind Wallet Handler ---
func (a *App) authRegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAuthError(w, http.StatusBadRequest, newAuthError("AUTH_INVALID_JSON", "validation", "invalid JSON"))
		return
	}

	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	if username == "" || password == "" {
		writeAuthError(w, http.StatusBadRequest, newAuthError("AUTH_MISSING_REGISTRATION_FIELDS", "validation", "username and password are required"))
		return
	}
	if len(password) < 6 {
		writeAuthError(w, http.StatusBadRequest, newAuthError("AUTH_WEAK_PASSWORD", "validation", "password must be at least 6 characters"))
		return
	}

	user, session, err := a.auth.Register(
		username,
		strings.TrimSpace(req.Email),
		password,
		strings.TrimSpace(req.WalletAddress),
		req.AutoWallet,
		req.ChainID,
		req.Nonce,
		req.Signature,
		r,
	)
	if err != nil {
		log.Printf("register failed for %q: %v", username, err)
		status := http.StatusBadRequest
		if isAuthErrorCode(err, "AUTH_USERNAME_TAKEN") || isAuthErrorCode(err, "AUTH_EMAIL_TAKEN") || isAuthErrorCode(err, "AUTH_WALLET_ALREADY_BOUND") {
			status = http.StatusConflict
		}
		writeAuthError(w, status, err)
		return
	}

	setSessionCookie(w, session.ID)
	log.Printf("register success: user=%s handle=%s", user.ID, user.Handle)
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "data": authSessionResponseFromUser(user)})
}

// --- Bind Challenge Handler ---
func (a *App) authBindChallengeHandler(w http.ResponseWriter, r *http.Request) {
	var req BindChallengeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAuthError(w, http.StatusBadRequest, newAuthError("AUTH_INVALID_JSON", "validation", "invalid JSON"))
		return
	}

	chainID := req.ChainID
	if chainID == 0 {
		if a.chainID != nil {
			chainID = a.chainID.Int64()
		} else {
			chainID = 1
		}
	}

	challenge, err := a.auth.CreateChallenge(req.WalletAddress, chainID, r)
	if err != nil {
		writeAuthError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": challenge})
}

func (a *App) optionalAuthenticatedUser(r *http.Request) (*SocialUser, error) {
	return a.auth.UserFromRequest(r)
}

func (a *App) requireAuthenticatedUser(w http.ResponseWriter, r *http.Request) (*SocialUser, bool) {
	user, err := a.optionalAuthenticatedUser(r)
	if err != nil {
		clearSessionCookie(w)
		writeAuthError(w, http.StatusUnauthorized, err)
		return nil, false
	}
	if user == nil {
		writeAuthError(w, http.StatusUnauthorized, newAuthError("AUTH_SESSION_REQUIRED", "session", errAuthSessionMissing.Error()))
		return nil, false
	}
	return user, true
}

func (s *AuthService) CreateChallenge(address string, chainID int64, r *http.Request) (AuthChallenge, error) {
	normalizedAddress, err := normalizeWalletAddress(address)
	if err != nil {
		return AuthChallenge{}, err
	}

	nonce, err := randomToken(16)
	if err != nil {
		return AuthChallenge{}, fmt.Errorf("generate nonce: %w", err)
	}

	issuedAt := time.Now().UTC()
	expiresAt := issuedAt.Add(authChallengeTTL)

	challenge := AuthChallenge{
		Nonce:     nonce,
		Address:   normalizedAddress,
		Message:   buildWalletSignMessage(normalizedAddress, chainID, requestOriginURI(r), issuedAt, expiresAt, nonce),
		ChainID:   chainID,
		IssuedAt:  issuedAt.Format(time.RFC3339),
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}

	if err := s.saveChallenge(challenge); err != nil {
		return AuthChallenge{}, err
	}

	return challenge, nil
}

func (s *AuthService) VerifyWalletLogin(address, nonce, signature string) (SocialUser, AuthSessionRecord, error) {
	challenge, err := s.getChallenge(strings.TrimSpace(nonce))
	if err != nil {
		return SocialUser{}, AuthSessionRecord{}, err
	}

	normalizedAddress, err := normalizeWalletAddress(address)
	if err != nil {
		return SocialUser{}, AuthSessionRecord{}, err
	}
	if !strings.EqualFold(normalizedAddress, challenge.Address) {
		return SocialUser{}, AuthSessionRecord{}, newAuthError("AUTH_WALLET_CHALLENGE_MISMATCH", "wallet", "wallet address does not match login challenge")
	}

	if err := verifyWalletSignature(challenge.Message, signature, challenge.Address); err != nil {
		return SocialUser{}, AuthSessionRecord{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	account, ok := s.findAccountByWalletNoLock(challenge.Address)
	if !ok && s.store != nil {
		storeAccount, storeOK, storeErr := s.store.FindAuthAccountByWallet(s.ctx, challenge.Address)
		if storeErr != nil {
			return SocialUser{}, AuthSessionRecord{}, storeErr
		}
		if storeOK {
			account = storeAccount
			ok = true
			s.accounts[account.ID] = account
		}
	}
	if !ok {
		return SocialUser{}, AuthSessionRecord{}, newAuthError("AUTH_WALLET_NOT_BOUND", "wallet", "wallet is not bound to any account, please register first")
	}
	if strings.TrimSpace(account.UserID) == "" {
		return SocialUser{}, AuthSessionRecord{}, newAuthError("AUTH_ACCOUNT_PROFILE_MISSING", "account", "wallet account is missing linked profile")
	}

	user, err := s.getUserByIDNoLock(account.UserID)
	if err != nil {
		return SocialUser{}, AuthSessionRecord{}, newAuthError("AUTH_LINKED_USER_NOT_FOUND", "account", "linked user not found")
	}

	session, err := s.createSessionForUserNoLock(user.ID, account.Wallet)
	if err != nil {
		return SocialUser{}, AuthSessionRecord{}, err
	}

	_ = s.deleteChallenge(challenge.Nonce)
	return *user, session, nil
}

// --- Password Login Logic ---
func (s *AuthService) PasswordLogin(identifier, password string) (SocialUser, AuthSessionRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var account *Account
	lowerIdentifier := strings.ToLower(identifier)
	for _, acc := range s.accounts {
		if strings.ToLower(acc.Username) == lowerIdentifier || (acc.Email != "" && strings.ToLower(acc.Email) == lowerIdentifier) {
			copy := acc
			account = &copy
			break
		}
	}
	if account == nil && s.store != nil {
		accounts, err := s.store.ListAuthAccounts(s.ctx)
		if err == nil {
			for _, acc := range accounts {
				if strings.ToLower(acc.Username) == lowerIdentifier || (acc.Email != "" && strings.ToLower(acc.Email) == lowerIdentifier) {
					copy := acc
					account = &copy
					s.accounts[acc.ID] = acc
					break
				}
			}
		}
	}
	if account == nil {
		return SocialUser{}, AuthSessionRecord{}, newAuthError("AUTH_ACCOUNT_NOT_FOUND", "credentials", "account not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)); err != nil {
		return SocialUser{}, AuthSessionRecord{}, newAuthError("AUTH_INVALID_PASSWORD", "credentials", "invalid password")
	}
	if strings.TrimSpace(account.Wallet) == "" {
		return SocialUser{}, AuthSessionRecord{}, newAuthError("AUTH_WALLET_REQUIRED", "account", "account has no bound wallet")
	}
	if strings.TrimSpace(account.UserID) == "" {
		return SocialUser{}, AuthSessionRecord{}, newAuthError("AUTH_ACCOUNT_PROFILE_MISSING", "account", "account has no linked profile")
	}

	user, err := s.getUserByIDNoLock(account.UserID)
	if err != nil {
		return SocialUser{}, AuthSessionRecord{}, newAuthError("AUTH_LINKED_USER_NOT_FOUND", "account", "linked user not found")
	}

	session, err := s.createSessionForUserNoLock(user.ID, account.Wallet)
	if err != nil {
		return SocialUser{}, AuthSessionRecord{}, err
	}

	return *user, session, nil
}

// --- Register + Bind Wallet Logic ---
func (s *AuthService) Register(username, email, password, walletAddress string, autoWallet bool, chainID int64, nonce, signature string, r *http.Request) (SocialUser, AuthSessionRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	trimmedWalletAddress := strings.TrimSpace(walletAddress)
	useAutoWallet := autoWallet || trimmedWalletAddress == ""

	// Check username uniqueness
	lowerUsername := strings.ToLower(username)
	for _, acc := range s.accounts {
		if strings.ToLower(acc.Username) == lowerUsername {
			return SocialUser{}, AuthSessionRecord{}, newAuthError("AUTH_USERNAME_TAKEN", "conflict", "username already taken")
		}
	}

	// Check email uniqueness if provided
	if email != "" {
		lowerEmail := strings.ToLower(email)
		for _, acc := range s.accounts {
			if acc.Email != "" && strings.ToLower(acc.Email) == lowerEmail {
				return SocialUser{}, AuthSessionRecord{}, newAuthError("AUTH_EMAIL_TAKEN", "conflict", "email already registered")
			}
		}
	}

	var (
		normalizedWallet string
		err              error
	)
	if useAutoWallet {
		autoGeneratedWallet, err := s.createUniqueAutoWalletNoLock()
		if err != nil {
			return SocialUser{}, AuthSessionRecord{}, err
		}
		normalizedWallet = autoGeneratedWallet
	} else {
		if strings.TrimSpace(nonce) == "" || strings.TrimSpace(signature) == "" {
			return SocialUser{}, AuthSessionRecord{}, newAuthError("AUTH_BIND_SIGNATURE_REQUIRED", "validation", "wallet binding nonce and signature are required")
		}

		normalizedWallet, err = normalizeWalletAddress(trimmedWalletAddress)
		if err != nil {
			return SocialUser{}, AuthSessionRecord{}, err
		}
	}

	// Check wallet uniqueness
	for _, acc := range s.accounts {
		if strings.EqualFold(acc.Wallet, normalizedWallet) {
			return SocialUser{}, AuthSessionRecord{}, newAuthError("AUTH_WALLET_ALREADY_BOUND", "conflict", "wallet already bound to another account")
		}
	}

	if !useAutoWallet {
		// Verify bind challenge signature
		challenge, err := s.getChallengeNoLock(nonce)
		if err != nil {
			return SocialUser{}, AuthSessionRecord{}, fmt.Errorf("bind challenge: %w", err)
		}
		if !strings.EqualFold(normalizedWallet, challenge.Address) {
			return SocialUser{}, AuthSessionRecord{}, newAuthError("AUTH_BIND_CHALLENGE_MISMATCH", "wallet", "wallet address does not match bind challenge")
		}
		if err := verifyWalletSignature(challenge.Message, signature, challenge.Address); err != nil {
			return SocialUser{}, AuthSessionRecord{}, err
		}
		_ = s.deleteChallengeNoLock(nonce)
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return SocialUser{}, AuthSessionRecord{}, fmt.Errorf("hash password: %w", err)
	}

	// Create social user + auth account in one DB transaction when PostgreSQL is active
	var user SocialUser
	now := time.Now().UTC()
	accountID := "acc_" + fmt.Sprintf("%d", now.UnixNano())
	account := Account{
		ID:           accountID,
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		Wallet:       normalizedWallet,
		Status:       "active",
		CreatedAt:    now.Format(time.RFC3339),
		UpdatedAt:    now.Format(time.RFC3339),
	}
	if pgStore, ok := s.store.(*PostgresUserStore); ok {
		user, account, err = pgStore.CreateWithAccount(s.ctx, CreateUserRequest{
			Handle:      username,
			DisplayName: username,
			Bio:         "MoleSociety member",
			Instance:    "vault.social",
			Wallet:      normalizedWallet,
		}, account)
	} else if s.store != nil {
		user, err = s.store.Create(s.ctx, CreateUserRequest{
			Handle:      username,
			DisplayName: username,
			Bio:         "MoleSociety member",
			Instance:    "vault.social",
			Wallet:      normalizedWallet,
		})
		account.UserID = user.ID
	} else {
		user, err = s.social.CreateUser(CreateUserRequest{
			Handle:      username,
			DisplayName: username,
			Bio:         "MoleSociety member",
			Instance:    "vault.social",
			Wallet:      normalizedWallet,
		})
		account.UserID = user.ID
	}
	if err != nil {
		return SocialUser{}, AuthSessionRecord{}, fmt.Errorf("create user: %w", err)
	}

	s.accounts[accountID] = account
	if _, ok := s.store.(*PostgresUserStore); !ok {
		s.persistAccountsLocked()
	}

	// Create session
	session, err := s.createSessionForUserNoLock(user.ID, normalizedWallet)
	if err != nil {
		return SocialUser{}, AuthSessionRecord{}, err
	}

	storedUser, err := s.getUserByIDNoLock(user.ID)
	if err != nil {
		return SocialUser{}, AuthSessionRecord{}, fmt.Errorf("reload user: %w", err)
	}

	return *storedUser, session, nil
}

func (s *AuthService) createUniqueAutoWalletNoLock() (string, error) {
	for i := 0; i < 5; i++ {
		privateKey, err := crypto.GenerateKey()
		if err != nil {
			return "", newAuthError("AUTH_AUTO_WALLET_FAILED", "wallet", "failed to create wallet")
		}
		address := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
		if _, exists := s.findAccountByWalletNoLock(address); !exists {
			return address, nil
		}
	}
	return "", newAuthError("AUTH_AUTO_WALLET_FAILED", "wallet", "failed to create wallet")
}

func (s *AuthService) findAccountByWalletNoLock(wallet string) (Account, bool) {
	for _, acc := range s.accounts {
		if strings.EqualFold(strings.TrimSpace(acc.Wallet), strings.TrimSpace(wallet)) {
			return acc, true
		}
	}
	return Account{}, false
}

func (s *AuthService) createSessionForUserNoLock(userID, wallet string) (AuthSessionRecord, error) {
	now := time.Now().UTC()
	sessionID, err := randomToken(32)
	if err != nil {
		return AuthSessionRecord{}, fmt.Errorf("generate session: %w", err)
	}

	session := AuthSessionRecord{
		ID:        sessionID,
		UserID:    userID,
		Address:   wallet,
		CreatedAt: now.Format(time.RFC3339),
		ExpiresAt: now.Add(authSessionTTL).Format(time.RFC3339),
	}

	if err := s.saveSession(session); err != nil {
		return AuthSessionRecord{}, err
	}

	return session, nil
}

func (s *AuthService) UserFromRequest(r *http.Request) (*SocialUser, error) {
	cookie, err := r.Cookie(authSessionCookieName)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return nil, nil
		}
		return nil, errAuthSessionInvalid
	}

	sessionID := strings.TrimSpace(cookie.Value)
	if sessionID == "" {
		return nil, nil
	}

	session, err := s.getSession(sessionID)
	if err != nil {
		if errors.Is(err, errAuthSessionMissing) || errors.Is(err, errAuthSessionInvalid) {
			return nil, errAuthSessionInvalid
		}
		return nil, err
	}

	user, err := s.getUserByIDNoLock(session.UserID)
	if err != nil {
		_ = s.DeleteSession(sessionID)
		return nil, errAuthSessionInvalid
	}

	return user, nil
}

func (s *AuthService) DeleteSession(sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil
	}

	if s.rdb != nil {
		return s.rdb.Del(s.ctx, authSessionKey(sessionID)).Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
	return nil
}

func (s *AuthService) saveChallenge(challenge AuthChallenge) error {
	if s.rdb != nil {
		return saveJSONWithTTL(s.ctx, s.rdb, authChallengeKey(challenge.Nonce), challenge, authChallengeTTL)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.challenges[challenge.Nonce] = challenge
	return nil
}

func (s *AuthService) getChallenge(nonce string) (AuthChallenge, error) {
	if strings.TrimSpace(nonce) == "" {
		return AuthChallenge{}, errors.New("nonce is required")
	}

	if s.rdb != nil {
		var challenge AuthChallenge
		if err := loadJSON(s.ctx, s.rdb, authChallengeKey(nonce), &challenge); err != nil {
			if errors.Is(err, redis.Nil) {
				return AuthChallenge{}, errors.New("login challenge expired or missing")
			}
			return AuthChallenge{}, err
		}
		if isExpired(challenge.ExpiresAt) {
			_ = s.deleteChallenge(nonce)
			return AuthChallenge{}, errors.New("login challenge expired or missing")
		}
		return challenge, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	challenge, ok := s.challenges[nonce]
	if !ok || isExpired(challenge.ExpiresAt) {
		delete(s.challenges, nonce)
		return AuthChallenge{}, errors.New("login challenge expired or missing")
	}
	return challenge, nil
}

func (s *AuthService) deleteChallenge(nonce string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleteChallengeNoLock(nonce)
}

func (s *AuthService) deleteChallengeNoLock(nonce string) error {
	if s.rdb != nil {
		return s.rdb.Del(s.ctx, authChallengeKey(nonce)).Err()
	}

	delete(s.challenges, nonce)
	return nil
}

func (s *AuthService) getChallengeNoLock(nonce string) (AuthChallenge, error) {
	if strings.TrimSpace(nonce) == "" {
		return AuthChallenge{}, errors.New("nonce is required")
	}

	if s.rdb != nil {
		var challenge AuthChallenge
		if err := loadJSON(s.ctx, s.rdb, authChallengeKey(nonce), &challenge); err != nil {
			if errors.Is(err, redis.Nil) {
				return AuthChallenge{}, errors.New("bind challenge expired or missing")
			}
			return AuthChallenge{}, err
		}
		if isExpired(challenge.ExpiresAt) {
			_ = s.deleteChallengeNoLock(nonce)
			return AuthChallenge{}, errors.New("bind challenge expired or missing")
		}
		return challenge, nil
	}

	challenge, ok := s.challenges[nonce]
	if !ok || isExpired(challenge.ExpiresAt) {
		delete(s.challenges, nonce)
		return AuthChallenge{}, errors.New("bind challenge expired or missing")
	}
	return challenge, nil
}

func (s *AuthService) saveSession(session AuthSessionRecord) error {
	if s.rdb != nil {
		return saveJSONWithTTL(s.ctx, s.rdb, authSessionKey(session.ID), session, authSessionTTL)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return nil
}

func (s *AuthService) getSession(sessionID string) (AuthSessionRecord, error) {
	if strings.TrimSpace(sessionID) == "" {
		return AuthSessionRecord{}, errAuthSessionMissing
	}

	if s.rdb != nil {
		var session AuthSessionRecord
		if err := loadJSON(s.ctx, s.rdb, authSessionKey(sessionID), &session); err != nil {
			if errors.Is(err, redis.Nil) {
				return AuthSessionRecord{}, errAuthSessionInvalid
			}
			return AuthSessionRecord{}, err
		}
		if isExpired(session.ExpiresAt) {
			_ = s.DeleteSession(sessionID)
			return AuthSessionRecord{}, errAuthSessionInvalid
		}
		return session, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[sessionID]
	if !ok || isExpired(session.ExpiresAt) {
		delete(s.sessions, sessionID)
		return AuthSessionRecord{}, errAuthSessionInvalid
	}
	return session, nil
}

func saveJSONWithTTL(ctx context.Context, rdb *redis.Client, key string, payload any, ttl time.Duration) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return rdb.Set(ctx, key, raw, ttl).Err()
}

func loadJSON(ctx context.Context, rdb *redis.Client, key string, target any) error {
	raw, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), target)
}

func authChallengeKey(nonce string) string {
	return "auth:challenge:" + nonce
}

func authSessionKey(sessionID string) string {
	return "auth:session:" + sessionID
}

func authSessionResponseFromUser(user SocialUser) AuthSessionResponse {
	return AuthSessionResponse{
		ID:          user.ID,
		Handle:      user.Handle,
		DisplayName: user.DisplayName,
		Instance:    user.Instance,
		Bio:         user.Bio,
		AvatarURL:   user.AvatarURL,
		Wallet:      user.Wallet,
	}
}

func normalizeWalletAddress(address string) (string, error) {
	trimmed := strings.TrimSpace(address)
	if !common.IsHexAddress(trimmed) {
		return "", errors.New("invalid wallet address")
	}
	return common.HexToAddress(trimmed).Hex(), nil
}

func verifyWalletSignature(message, signature, expectedAddress string) error {
	decoded := common.FromHex(strings.TrimSpace(signature))
	if len(decoded) != crypto.SignatureLength {
		return errors.New("invalid wallet signature")
	}

	signatureCopy := append([]byte(nil), decoded...)
	if signatureCopy[crypto.RecoveryIDOffset] >= 27 {
		signatureCopy[crypto.RecoveryIDOffset] -= 27
	}
	if signatureCopy[crypto.RecoveryIDOffset] > 1 {
		return errors.New("invalid wallet signature")
	}

	hash := accounts.TextHash([]byte(message))
	publicKey, err := crypto.SigToPub(hash, signatureCopy)
	if err != nil {
		return errors.New("failed to verify wallet signature")
	}

	recovered := crypto.PubkeyToAddress(*publicKey).Hex()
	if !strings.EqualFold(recovered, expectedAddress) {
		return errors.New("wallet signature does not match address")
	}

	return nil
}

func buildWalletSignMessage(address string, chainID int64, uri string, issuedAt time.Time, expiresAt time.Time, nonce string) string {
	return fmt.Sprintf(
		"MoleSociety wants you to sign in with your wallet:\n%s\n\nSign in to MoleSociety.\n\nURI: %s\nVersion: 1\nChain ID: %d\nNonce: %s\nIssued At: %s\nExpiration Time: %s",
		address,
		uri,
		chainID,
		nonce,
		issuedAt.Format(time.RFC3339),
		expiresAt.Format(time.RFC3339),
	)
}

func requestOriginURI(r *http.Request) string {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin != "" {
		return origin
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	host := strings.TrimSpace(r.Host)
	if host == "" {
		host = "localhost"
	}
	if parsed, err := url.Parse(fmt.Sprintf("%s://%s", scheme, host)); err == nil {
		return parsed.String()
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}

func randomToken(byteLength int) (string, error) {
	buffer := make([]byte, byteLength)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func isExpired(timestamp string) bool {
	if strings.TrimSpace(timestamp) == "" {
		return true
	}
	expiresAt, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return true
	}
	return time.Now().UTC().After(expiresAt)
}

func setSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     authSessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: sessionCookieSameSite(),
		Secure:   sessionCookieSecure(),
		MaxAge:   int(authSessionTTL.Seconds()),
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authSessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: sessionCookieSameSite(),
		Secure:   sessionCookieSecure(),
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func sessionCookieSameSite() http.SameSite {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("COOKIE_SAMESITE")), "none") {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

func sessionCookieSecure() bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("COOKIE_SAMESITE")), "none") {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(os.Getenv("COOKIE_SECURE")), "true")
}

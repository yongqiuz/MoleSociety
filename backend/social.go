package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
)

type SocialUser struct {
	ID          string `json:"id"`
	Handle      string `json:"handle"`
	DisplayName string `json:"displayName"`
	Bio         string `json:"bio"`
	Instance    string `json:"instance"`
	Wallet      string `json:"wallet"`
	AvatarURL   string `json:"avatarUrl"`
	Followers   int    `json:"followers"`
	Following   int    `json:"following"`
	CreatedAt   string `json:"createdAt"`
}

type MediaAsset struct {
	ID         string `json:"id"`
	OwnerID    string `json:"ownerId"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	URL        string `json:"url"`
	StorageURI string `json:"storageUri"`
	CID        string `json:"cid"`
	SizeBytes  int64  `json:"sizeBytes"`
	Status     string `json:"status"`
	CreatedAt  string `json:"createdAt"`
}

type PostMedia struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	URL        string `json:"url"`
	Kind       string `json:"kind"`
	StorageURI string `json:"storageUri"`
	CID        string `json:"cid"`
}

type SocialPost struct {
	ID             string      `json:"id"`
	AuthorID       string      `json:"authorId"`
	AuthorHandle   string      `json:"authorHandle"`
	AuthorName     string      `json:"authorName"`
	Instance       string      `json:"instance"`
	Kind           string      `json:"kind"`
	Content        string      `json:"content"`
	Visibility     string      `json:"visibility"`
	StorageURI     string      `json:"storageUri"`
	AttestationURI string      `json:"attestationUri"`
	Tags           []string    `json:"tags"`
	Media          []PostMedia `json:"media"`
	ParentPostID   string      `json:"parentPostId,omitempty"`
	RootPostID     string      `json:"rootPostId,omitempty"`
	ReplyDepth     int         `json:"replyDepth,omitempty"`
	Replies        int         `json:"replies"`
	Boosts         int         `json:"boosts"`
	Likes          int         `json:"likes"`
	CreatedAt      string      `json:"createdAt"`
}

type PostThread struct {
	Post      SocialPost   `json:"post"`
	Ancestors []SocialPost `json:"ancestors"`
	Replies   []SocialPost `json:"replies"`
}

type ChatMessage struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversationId"`
	SenderID       string `json:"senderId"`
	SenderHandle   string `json:"senderHandle"`
	Body           string `json:"body"`
	CreatedAt      string `json:"createdAt"`
}

type Conversation struct {
	ID             string        `json:"id"`
	Title          string        `json:"title"`
	ParticipantIDs []string      `json:"participantIds"`
	Encrypted      bool          `json:"encrypted"`
	Messages       []ChatMessage `json:"messages"`
	UpdatedAt      string        `json:"updatedAt"`
}

type FederationInstance struct {
	Name    string `json:"name"`
	Focus   string `json:"focus"`
	Members string `json:"members"`
	Latency string `json:"latency"`
	Status  string `json:"status"`
}

type SocialStats struct {
	Users         int `json:"users"`
	Posts         int `json:"posts"`
	MediaAssets   int `json:"mediaAssets"`
	Conversations int `json:"conversations"`
}

type BootstrapPayload struct {
	CurrentUser   *SocialUser          `json:"currentUser,omitempty"`
	Stats         SocialStats          `json:"stats"`
	Feed          []SocialPost         `json:"feed"`
	Users         []SocialUser         `json:"users"`
	Media         []MediaAsset         `json:"media"`
	Conversations []Conversation       `json:"conversations"`
	Instances     []FederationInstance `json:"instances"`
}

type CreateUserRequest struct {
	Handle      string `json:"handle"`
	DisplayName string `json:"displayName"`
	Bio         string `json:"bio"`
	Instance    string `json:"instance"`
	Wallet      string `json:"wallet"`
	AvatarURL   string `json:"avatarUrl"`
}

type CreateMediaRequest struct {
	OwnerID    string `json:"ownerId"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	URL        string `json:"url"`
	StorageURI string `json:"storageUri"`
	CID        string `json:"cid"`
	SizeBytes  int64  `json:"sizeBytes"`
	Status     string `json:"status"`
}

type CreatePostRequest struct {
	AuthorID       string   `json:"authorId"`
	Kind           string   `json:"kind"`
	Content        string   `json:"content"`
	Visibility     string   `json:"visibility"`
	StorageURI     string   `json:"storageUri"`
	AttestationURI string   `json:"attestationUri"`
	Tags           []string `json:"tags"`
	MediaIDs       []string `json:"mediaIds"`
	ParentPostID   string   `json:"parentPostId"`
	RootPostID     string   `json:"rootPostId"`
}

type CreateConversationRequest struct {
	Title          string   `json:"title"`
	ParticipantIDs []string `json:"participantIds"`
	Encrypted      bool     `json:"encrypted"`
}

type CreateMessageRequest struct {
	SenderID string `json:"senderId"`
	Body     string `json:"body"`
}

type SocialService struct {
	ctx           context.Context
	rdb           *redis.Client
	mu            sync.RWMutex
	users         []SocialUser
	posts         []SocialPost
	media         []MediaAsset
	conversations []Conversation
	instances     []FederationInstance
}

func NewSocialService(ctx context.Context, rdb *redis.Client) *SocialService {
	service := &SocialService{ctx: ctx, rdb: rdb}
	service.load()
	return service
}

func (s *SocialService) load() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.rdb != nil && s.loadSnapshotsFromRedis() {
		return
	}

	s.seedDefaults()
	s.persistLocked()
}

func (s *SocialService) loadSnapshotsFromRedis() bool {
	var users []SocialUser
	var posts []SocialPost
	var media []MediaAsset
	var conversations []Conversation
	var instances []FederationInstance

	loaders := []struct {
		key    string
		target any
	}{
		{key: "social:snapshot:users", target: &users},
		{key: "social:snapshot:posts", target: &posts},
		{key: "social:snapshot:media", target: &media},
		{key: "social:snapshot:conversations", target: &conversations},
		{key: "social:snapshot:instances", target: &instances},
	}

	for _, loader := range loaders {
		raw, err := s.rdb.Get(s.ctx, loader.key).Result()
		if err != nil || strings.TrimSpace(raw) == "" {
			return false
		}
		if err := json.Unmarshal([]byte(raw), loader.target); err != nil {
			return false
		}
	}

	s.users = users
	s.posts = posts
	s.media = media
	s.conversations = conversations
	s.instances = instances
	s.normalizePostsLocked()
	return true
}

func (s *SocialService) seedDefaults() {
	now := time.Now().UTC()
	s.instances = []FederationInstance{
		{Name: "vault.social", Focus: "创作者主权与链上身份", Members: "12.4k", Latency: "43 ms", Status: "healthy"},
		{Name: "readers.polkadot", Focus: "阅读社群与数字馆藏", Members: "8.9k", Latency: "51 ms", Status: "healthy"},
		{Name: "relay.zone", Focus: "跨实例消息转发", Members: "3.1k", Latency: "37 ms", Status: "healthy"},
		{Name: "storage.zone", Focus: "媒体与永续资源镜像", Members: "5.7k", Latency: "49 ms", Status: "healthy"},
	}

	s.users = []SocialUser{
		{
			ID:          "user_archive",
			Handle:      "@archive",
			DisplayName: "Whale Archive",
			Bio:         "为创作者提供永久内容归档与链上身份锚定。",
			Instance:    "vault.social",
			Wallet:      "0xa18f...3c92",
			AvatarURL:   "https://picsum.photos/seed/archive/128/128",
			Followers:   1284,
			Following:   312,
			CreatedAt:   now.Add(-48 * time.Hour).Format(time.RFC3339),
		},
		{
			ID:          "user_librarian",
			Handle:      "@librarian",
			DisplayName: "Node Librarian",
			Bio:         "把书籍确权、媒体存储和去中心化社交连接在一起。",
			Instance:    "readers.polkadot",
			Wallet:      "0x78fe...12ab",
			AvatarURL:   "https://picsum.photos/seed/librarian/128/128",
			Followers:   932,
			Following:   221,
			CreatedAt:   now.Add(-36 * time.Hour).Format(time.RFC3339),
		},
		{
			ID:          "user_fedilab",
			Handle:      "@fedilab",
			DisplayName: "Open Federation Lab",
			Bio:         "探索 ActivityPub、实时会话和多实例协作。",
			Instance:    "relay.zone",
			Wallet:      "0x95bc...09ee",
			AvatarURL:   "https://picsum.photos/seed/fedilab/128/128",
			Followers:   1650,
			Following:   415,
			CreatedAt:   now.Add(-24 * time.Hour).Format(time.RFC3339),
		},
	}

	s.media = []MediaAsset{
		{
			ID:         "media_manifesto",
			OwnerID:    "user_archive",
			Name:       "genesis-manifesto.png",
			Kind:       "image",
			URL:        "https://picsum.photos/seed/manifesto/1200/800",
			StorageURI: "ar://7xv91manifesto",
			CID:        "bafybeih7f4manifesto",
			SizeBytes:  2400000,
			Status:     "mirrored",
			CreatedAt:  now.Add(-20 * time.Hour).Format(time.RFC3339),
		},
		{
			ID:         "media_space",
			OwnerID:    "user_librarian",
			Name:       "weekly-space.mp4",
			Kind:       "video",
			URL:        "https://picsum.photos/seed/space/1200/800",
			StorageURI: "ar://weekly-space",
			CID:        "ar://space-video",
			SizeBytes:  84000000,
			Status:     "stored",
			CreatedAt:  now.Add(-12 * time.Hour).Format(time.RFC3339),
		},
	}

	s.posts = []SocialPost{
		{
			ID:             "post_archive",
			AuthorID:       "user_archive",
			AuthorHandle:   "@archive",
			AuthorName:     "Whale Archive",
			Instance:       "vault.social",
			Kind:           "post",
			Content:        "把“每本书一个 NFT 身份”的思路升级成社交协议之后，内容、关系和媒体都应该拥有可迁移、可验证、可存档的数字主权。",
			Visibility:     "public",
			StorageURI:     "ar://post-archive",
			AttestationURI: "attestation://bookproof/0xa18f...3c92",
			Tags:           []string{"去中心化社交", "数字主权", "链上身份"},
			Boosts:         31,
			Likes:          88,
			CreatedAt:      now.Add(-2 * time.Hour).Format(time.RFC3339),
		},
		{
			ID:             "post_librarian",
			AuthorID:       "user_librarian",
			AuthorHandle:   "@librarian",
			AuthorName:     "Node Librarian",
			Instance:       "readers.polkadot",
			Kind:           "post",
			Content:        "新媒体上传已同步到 Arweave 与 IPFS 双存储层。只要内容哈希一致，前端、实例、检索器都能独立重建同一份帖子上下文。",
			Visibility:     "public",
			StorageURI:     "ar://post-librarian",
			AttestationURI: "storage://arweave/S1NfXo2...8vdP",
			Tags:           []string{"Arweave", "IPFS", "永久媒体"},
			Media: []PostMedia{
				{
					ID:         "media_manifesto",
					Name:       "genesis-manifesto.png",
					URL:        "https://picsum.photos/seed/manifesto/1200/800",
					Kind:       "image",
					StorageURI: "ar://7xv91manifesto",
					CID:        "bafybeih7f4manifesto",
				},
			},
			Boosts:    21,
			Likes:     64,
			CreatedAt: now.Add(-90 * time.Minute).Format(time.RFC3339),
		},
		{
			ID:             "post_fedilab",
			AuthorID:       "user_fedilab",
			AuthorHandle:   "@fedilab",
			AuthorName:     "Open Federation Lab",
			Instance:       "relay.zone",
			Kind:           "post",
			Content:        "接下来要把当前的 relay server 从“扫码 mint”升级为 ActivityPub + 媒体索引 + 实时会话网关，让不同实例之间的关注、转发和聊天真正互通。",
			Visibility:     "public",
			StorageURI:     "ar://post-fedilab",
			AttestationURI: "relay://federation/upgrade-plan",
			Tags:           []string{"ActivityPub", "实时聊天", "Go Relay"},
			Boosts:         55,
			Likes:          102,
			CreatedAt:      now.Add(-45 * time.Minute).Format(time.RFC3339),
		},
		{
			ID:             "reply_archive_1",
			AuthorID:       "user_librarian",
			AuthorHandle:   "@librarian",
			AuthorName:     "Node Librarian",
			Instance:       "readers.polkadot",
			Kind:           "reply",
			Content:        "统一 posts 之后，评论不再是孤立记录，线程、引用和排序都能复用同一套内容基础设施。",
			Visibility:     "public",
			StorageURI:     "ar://reply-archive-1",
			AttestationURI: "attestation://reply/archive/1",
			Tags:           []string{"线程模型", "统一内容"},
			ParentPostID:   "post_archive",
			RootPostID:     "post_archive",
			ReplyDepth:     1,
			Likes:          16,
			CreatedAt:      now.Add(-105 * time.Minute).Format(time.RFC3339),
		},
		{
			ID:             "reply_archive_2",
			AuthorID:       "user_fedilab",
			AuthorHandle:   "@fedilab",
			AuthorName:     "Open Federation Lab",
			Instance:       "relay.zone",
			Kind:           "reply",
			Content:        "后面接 ActivityPub 的时候，也能直接把 reply 当作 Note 的一种关系分支来处理。",
			Visibility:     "public",
			StorageURI:     "ar://reply-archive-2",
			AttestationURI: "attestation://reply/archive/2",
			Tags:           []string{"ActivityPub", "回复"},
			ParentPostID:   "post_archive",
			RootPostID:     "post_archive",
			ReplyDepth:     1,
			Likes:          12,
			CreatedAt:      now.Add(-95 * time.Minute).Format(time.RFC3339),
		},
		{
			ID:             "reply_librarian_1",
			AuthorID:       "user_archive",
			AuthorHandle:   "@archive",
			AuthorName:     "Whale Archive",
			Instance:       "vault.social",
			Kind:           "reply",
			Content:        "媒体与正文拆开存储后，评论区也能只引用媒体哈希而不是复制整块内容。",
			Visibility:     "public",
			StorageURI:     "ar://reply-librarian-1",
			AttestationURI: "attestation://reply/librarian/1",
			Tags:           []string{"媒体引用"},
			ParentPostID:   "post_librarian",
			RootPostID:     "post_librarian",
			ReplyDepth:     1,
			Likes:          9,
			CreatedAt:      now.Add(-70 * time.Minute).Format(time.RFC3339),
		},
		{
			ID:             "reply_fedilab_1",
			AuthorID:       "user_archive",
			AuthorHandle:   "@archive",
			AuthorName:     "Whale Archive",
			Instance:       "vault.social",
			Kind:           "reply",
			Content:        "这也意味着以后 quote、reply、thread view 都不用再扩第三张表。",
			Visibility:     "public",
			StorageURI:     "ar://reply-fedilab-1",
			AttestationURI: "attestation://reply/fedilab/1",
			Tags:           []string{"统一posts"},
			ParentPostID:   "post_fedilab",
			RootPostID:     "post_fedilab",
			ReplyDepth:     1,
			Likes:          18,
			CreatedAt:      now.Add(-35 * time.Minute).Format(time.RFC3339),
		},
	}

	s.conversations = []Conversation{
		{
			ID:             "conv_curator",
			Title:          "Archive Curator",
			ParticipantIDs: []string{"user_archive", "user_librarian"},
			Encrypted:      true,
			UpdatedAt:      now.Add(-15 * time.Minute).Format(time.RFC3339),
			Messages: []ChatMessage{
				{
					ID:             "msg_1",
					ConversationID: "conv_curator",
					SenderID:       "user_archive",
					SenderHandle:   "@archive",
					Body:           "我们把帖子正文上链证明，媒体放 Arweave，前端就能跨实例恢复。",
					CreatedAt:      now.Add(-20 * time.Minute).Format(time.RFC3339),
				},
				{
					ID:             "msg_2",
					ConversationID: "conv_curator",
					SenderID:       "user_librarian",
					SenderHandle:   "@librarian",
					Body:           "对，聊天部分我先做会话原型，后续再接 Matrix 或 libp2p。",
					CreatedAt:      now.Add(-18 * time.Minute).Format(time.RFC3339),
				},
			},
		},
	}

	s.normalizePostsLocked()
	s.sortLocked()
}

func (s *SocialService) persistLocked() {
	if s.rdb == nil {
		return
	}

	snapshots := map[string]any{
		"social:snapshot:users":         s.users,
		"social:snapshot:posts":         s.posts,
		"social:snapshot:media":         s.media,
		"social:snapshot:conversations": s.conversations,
		"social:snapshot:instances":     s.instances,
	}

	for key, value := range snapshots {
		raw, err := json.Marshal(value)
		if err != nil {
			continue
		}
		_ = s.rdb.Set(s.ctx, key, raw, 0).Err()
	}
}

func (s *SocialService) sortLocked() {
	sort.Slice(s.posts, func(i, j int) bool { return s.posts[i].CreatedAt > s.posts[j].CreatedAt })
	sort.Slice(s.media, func(i, j int) bool { return s.media[i].CreatedAt > s.media[j].CreatedAt })
	sort.Slice(s.conversations, func(i, j int) bool { return s.conversations[i].UpdatedAt > s.conversations[j].UpdatedAt })
}

func (s *SocialService) normalizePostsLocked() {
	for i := range s.posts {
		if strings.TrimSpace(s.posts[i].ParentPostID) != "" {
			s.posts[i].Kind = "reply"
			if strings.TrimSpace(s.posts[i].RootPostID) == "" {
				s.posts[i].RootPostID = s.posts[i].ParentPostID
			}
		} else if strings.TrimSpace(s.posts[i].Kind) == "" {
			s.posts[i].Kind = "post"
		}
	}
	s.refreshPostAggregatesLocked()
}

func (s *SocialService) refreshPostAggregatesLocked() {
	replyCounts := make(map[string]int, len(s.posts))
	for _, post := range s.posts {
		if strings.TrimSpace(post.ParentPostID) != "" {
			replyCounts[post.ParentPostID]++
		}
	}

	for i := range s.posts {
		s.posts[i].Replies = replyCounts[s.posts[i].ID]
	}
}

func (s *SocialService) Stats() SocialStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return SocialStats{
		Users:         len(s.users),
		Posts:         len(s.posts),
		MediaAssets:   len(s.media),
		Conversations: len(s.conversations),
	}
}

func (s *SocialService) Bootstrap(limit int, currentUserID string) BootstrapPayload {
	s.mu.RLock()
	defer s.mu.RUnlock()

	payload := BootstrapPayload{
		Stats:         SocialStats{Users: len(s.users), Posts: len(s.posts), MediaAssets: len(s.media), Conversations: len(s.conversations)},
		Feed:          append([]SocialPost(nil), sliceTopLevelPosts(s.posts, limit)...),
		Users:         append([]SocialUser(nil), s.users...),
		Media:         append([]MediaAsset(nil), sliceMedia(s.media, limit)...),
		Conversations: append([]Conversation(nil), sliceConversations(s.conversations, limit)...),
		Instances:     append([]FederationInstance(nil), s.instances...),
	}
	if strings.TrimSpace(currentUserID) != "" {
		if user, err := s.findUserLocked(currentUserID); err == nil {
			payload.CurrentUser = &user
			return payload
		}
	}

	if len(s.users) > 0 {
		user := s.users[0]
		payload.CurrentUser = &user
	}
	return payload
}

func (s *SocialService) ListInstances() []FederationInstance {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]FederationInstance(nil), s.instances...)
}

func (s *SocialService) ListUsers() []SocialUser {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]SocialUser(nil), s.users...)
}

func (s *SocialService) GetUser(id string) (*SocialUser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, user := range s.users {
		if user.ID == id {
			copyUser := user
			return &copyUser, nil
		}
	}
	return nil, errors.New("user not found: " + id)
}

func (s *SocialService) CreateUser(req CreateUserRequest) (SocialUser, error) {
	if strings.TrimSpace(req.Handle) == "" {
		return SocialUser{}, errors.New("handle is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	user := SocialUser{
		ID:          nextID("user"),
		Handle:      normalizeHandle(req.Handle),
		DisplayName: valueOrDefault(strings.TrimSpace(req.DisplayName), strings.TrimPrefix(normalizeHandle(req.Handle), "@")),
		Bio:         strings.TrimSpace(req.Bio),
		Instance:    valueOrDefault(strings.TrimSpace(req.Instance), "vault.social"),
		Wallet:      strings.TrimSpace(req.Wallet),
		AvatarURL:   strings.TrimSpace(req.AvatarURL),
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	s.users = append([]SocialUser{user}, s.users...)
	s.persistLocked()
	return user, nil
}

func (s *SocialService) EnsureWalletUser(wallet string) (SocialUser, error) {
	normalizedWallet := strings.TrimSpace(wallet)
	if normalizedWallet == "" {
		return SocialUser{}, errors.New("wallet address is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, user := range s.users {
		if strings.EqualFold(strings.TrimSpace(user.Wallet), normalizedWallet) {
			return user, nil
		}
	}

	shortWallet := strings.ToLower(strings.TrimPrefix(normalizedWallet, "0x"))
	handleSuffix := shortWallet
	if len(handleSuffix) > 8 {
		handleSuffix = handleSuffix[:8]
	}
	displaySuffix := shortWallet
	if len(displaySuffix) > 4 {
		displaySuffix = displaySuffix[len(displaySuffix)-4:]
	}

	user := SocialUser{
		ID:          nextID("user"),
		Handle:      normalizeHandle("mole-" + handleSuffix),
		DisplayName: "Mole " + strings.ToUpper(displaySuffix),
		Bio:         "Wallet-authenticated member",
		Instance:    "vault.social",
		Wallet:      normalizedWallet,
		AvatarURL:   "",
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	s.users = append([]SocialUser{user}, s.users...)
	s.persistLocked()
	return user, nil
}

func (s *SocialService) Feed(limit int) []SocialPost {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]SocialPost(nil), sliceTopLevelPosts(s.posts, limit)...)
}

func (s *SocialService) GetPost(id string) (*SocialPost, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, post := range s.posts {
		if post.ID == id {
			copyPost := post
			return &copyPost, nil
		}
	}
	return nil, errors.New("post not found: " + id)
}

func (s *SocialService) GetPostThread(id string, limit int) (*PostThread, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	post, index := s.findPostLocked(id)
	if index == -1 {
		return nil, errors.New("post not found: " + id)
	}

	ancestors := s.collectAncestorsLocked(post)
	replies := s.collectRepliesLocked(post.ID, limit)
	thread := &PostThread{
		Post:      post,
		Ancestors: ancestors,
		Replies:   replies,
	}
	return thread, nil
}

func (s *SocialService) ListReplies(postID string, limit int) ([]SocialPost, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, index := s.findPostLocked(postID); index == -1 {
		return nil, errors.New("post not found: " + postID)
	}

	return append([]SocialPost(nil), s.collectRepliesLocked(postID, limit)...), nil
}

func (s *SocialService) CreatePost(req CreatePostRequest) (SocialPost, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	author, err := s.findUserLocked(req.AuthorID)
	if err != nil {
		return SocialPost{}, err
	}

	parentPostID := strings.TrimSpace(req.ParentPostID)
	rootPostID := strings.TrimSpace(req.RootPostID)
	kind := valueOrDefault(strings.TrimSpace(req.Kind), "post")
	replyDepth := 0
	if parentPostID != "" {
		parentPost, parentIndex := s.findPostLocked(parentPostID)
		if parentIndex == -1 {
			return SocialPost{}, errors.New("parent post not found: " + parentPostID)
		}
		kind = "reply"
		if rootPostID == "" {
			rootPostID = valueOrDefault(parentPost.RootPostID, parentPost.ID)
		}
		replyDepth = parentPost.ReplyDepth + 1
	}

	attachments := make([]PostMedia, 0, len(req.MediaIDs))
	for _, mediaID := range req.MediaIDs {
		asset, mediaErr := s.findMediaLocked(mediaID)
		if mediaErr != nil {
			continue
		}
		attachments = append(attachments, PostMedia{
			ID:         asset.ID,
			Name:       asset.Name,
			URL:        asset.URL,
			Kind:       asset.Kind,
			StorageURI: asset.StorageURI,
			CID:        asset.CID,
		})
	}

	post := SocialPost{
		ID:             nextID("post"),
		AuthorID:       author.ID,
		AuthorHandle:   author.Handle,
		AuthorName:     author.DisplayName,
		Instance:       author.Instance,
		Kind:           kind,
		Content:        strings.TrimSpace(req.Content),
		Visibility:     valueOrDefault(strings.TrimSpace(req.Visibility), "public"),
		StorageURI:     strings.TrimSpace(req.StorageURI),
		AttestationURI: strings.TrimSpace(req.AttestationURI),
		Tags:           req.Tags,
		Media:          attachments,
		ParentPostID:   parentPostID,
		RootPostID:     rootPostID,
		ReplyDepth:     replyDepth,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}

	s.posts = append([]SocialPost{post}, s.posts...)
	s.refreshPostAggregatesLocked()
	s.sortLocked()
	s.persistLocked()
	return post, nil
}

func (s *SocialService) ListMedia(limit int) []MediaAsset {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]MediaAsset(nil), sliceMedia(s.media, limit)...)
}

func (s *SocialService) CreateMedia(req CreateMediaRequest) (MediaAsset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.findUserLocked(req.OwnerID); err != nil {
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

	s.media = append([]MediaAsset{asset}, s.media...)
	s.sortLocked()
	s.persistLocked()
	return asset, nil
}

func (s *SocialService) ListConversations(limit int) []Conversation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Conversation(nil), sliceConversations(s.conversations, limit)...)
}

func (s *SocialService) GetConversation(id string) (*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, conversation := range s.conversations {
		if conversation.ID == id {
			copyConversation := conversation
			return &copyConversation, nil
		}
	}
	return nil, errors.New("conversation not found: " + id)
}

func (s *SocialService) CreateConversation(req CreateConversationRequest) (Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, participantID := range req.ParticipantIDs {
		if _, err := s.findUserLocked(participantID); err != nil {
			return Conversation{}, err
		}
	}

	conversation := Conversation{
		ID:             nextID("conv"),
		Title:          valueOrDefault(strings.TrimSpace(req.Title), "New Conversation"),
		ParticipantIDs: append([]string(nil), req.ParticipantIDs...),
		Encrypted:      req.Encrypted,
		Messages:       []ChatMessage{},
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339),
	}

	s.conversations = append([]Conversation{conversation}, s.conversations...)
	s.sortLocked()
	s.persistLocked()
	return conversation, nil
}

func (s *SocialService) AddMessage(conversationID string, req CreateMessageRequest) (Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sender, err := s.findUserLocked(req.SenderID)
	if err != nil {
		return Conversation{}, err
	}

	for i := range s.conversations {
		if s.conversations[i].ID != conversationID {
			continue
		}

		isParticipant := false
		for _, participantID := range s.conversations[i].ParticipantIDs {
			if participantID == sender.ID {
				isParticipant = true
				break
			}
		}
		if !isParticipant {
			return Conversation{}, errors.New("sender is not a participant in this conversation")
		}

		message := ChatMessage{
			ID:             nextID("msg"),
			ConversationID: conversationID,
			SenderID:       sender.ID,
			SenderHandle:   sender.Handle,
			Body:           strings.TrimSpace(req.Body),
			CreatedAt:      time.Now().UTC().Format(time.RFC3339),
		}

		s.conversations[i].Messages = append(s.conversations[i].Messages, message)
		s.conversations[i].UpdatedAt = message.CreatedAt
		s.sortLocked()
		s.persistLocked()
		return s.conversations[i], nil
	}

	return Conversation{}, errors.New("conversation not found: " + conversationID)
}

func (s *SocialService) Distribution() []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	counts := map[string]int{}
	for _, user := range s.users {
		counts[user.Instance]++
	}

	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		result = append(result, map[string]any{
			"instance": key,
			"users":    counts[key],
		})
	}
	return result
}

func (a *App) bootstrapHandler(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 10)
	currentUser, _ := a.optionalAuthenticatedUser(r)
	currentUserID := ""
	if currentUser != nil {
		currentUserID = currentUser.ID
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": a.social.Bootstrap(limit, currentUserID)})
}

func (a *App) instancesHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": a.social.ListInstances()})
}

func (a *App) listUsersHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": a.social.ListUsers()})
}

func (a *App) createUserHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid JSON"})
		return
	}
	user, err := a.social.CreateUser(req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "data": user})
}

func (a *App) getUserHandler(w http.ResponseWriter, r *http.Request) {
	user, err := a.social.GetUser(mux.Vars(r)["id"])
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": user})
}

func (a *App) feedHandler(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 20)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": a.social.Feed(limit)})
}

func (a *App) createPostHandler(w http.ResponseWriter, r *http.Request) {
	authUser, ok := a.requireAuthenticatedUser(w, r)
	if !ok {
		return
	}

	var req CreatePostRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid JSON"})
		return
	}
	req.AuthorID = authUser.ID
	post, err := a.social.CreatePost(req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "data": post})
}

func (a *App) getPostHandler(w http.ResponseWriter, r *http.Request) {
	post, err := a.social.GetPost(mux.Vars(r)["id"])
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": post})
}

func (a *App) getPostThreadHandler(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 20)
	thread, err := a.social.GetPostThread(mux.Vars(r)["id"], limit)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": thread})
}

func (a *App) listPostRepliesHandler(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 20)
	replies, err := a.social.ListReplies(mux.Vars(r)["id"], limit)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": replies})
}

func (a *App) listMediaHandler(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 20)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": a.social.ListMedia(limit)})
}

func (a *App) createMediaHandler(w http.ResponseWriter, r *http.Request) {
	authUser, ok := a.requireAuthenticatedUser(w, r)
	if !ok {
		return
	}

	var req CreateMediaRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid JSON"})
		return
	}
	req.OwnerID = authUser.ID
	asset, err := a.social.CreateMedia(req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "data": asset})
}

func (a *App) listConversationsHandler(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 20)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": a.social.ListConversations(limit)})
}

func (a *App) createConversationHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateConversationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid JSON"})
		return
	}
	conversation, err := a.social.CreateConversation(req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "data": conversation})
}

func (a *App) getConversationHandler(w http.ResponseWriter, r *http.Request) {
	conversation, err := a.social.GetConversation(mux.Vars(r)["id"])
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": conversation})
}

func (a *App) addMessageHandler(w http.ResponseWriter, r *http.Request) {
	authUser, ok := a.requireAuthenticatedUser(w, r)
	if !ok {
		return
	}

	var req CreateMessageRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid JSON"})
		return
	}
	req.SenderID = authUser.ID
	conversation, err := a.social.AddMessage(mux.Vars(r)["id"], req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "data": conversation})
}

func (s *SocialService) findUserLocked(id string) (SocialUser, error) {
	for _, user := range s.users {
		if user.ID == id {
			return user, nil
		}
	}
	return SocialUser{}, errors.New("user not found: " + id)
}

func (s *SocialService) findMediaLocked(id string) (MediaAsset, error) {
	for _, asset := range s.media {
		if asset.ID == id {
			return asset, nil
		}
	}
	return MediaAsset{}, errors.New("media not found: " + id)
}

func (s *SocialService) findPostLocked(id string) (SocialPost, int) {
	for i, post := range s.posts {
		if post.ID == id {
			return post, i
		}
	}
	return SocialPost{}, -1
}

func (s *SocialService) collectAncestorsLocked(post SocialPost) []SocialPost {
	if strings.TrimSpace(post.ParentPostID) == "" {
		return nil
	}

	ancestors := []SocialPost{}
	currentParentID := post.ParentPostID
	for strings.TrimSpace(currentParentID) != "" {
		parent, index := s.findPostLocked(currentParentID)
		if index == -1 {
			break
		}
		ancestors = append([]SocialPost{parent}, ancestors...)
		currentParentID = parent.ParentPostID
	}
	return ancestors
}

func (s *SocialService) collectRepliesLocked(postID string, limit int) []SocialPost {
	children := make(map[string][]SocialPost, len(s.posts))
	for _, post := range s.posts {
		parentID := strings.TrimSpace(post.ParentPostID)
		if parentID == "" {
			continue
		}
		children[parentID] = append(children[parentID], post)
	}

	for parentID := range children {
		sort.Slice(children[parentID], func(i, j int) bool {
			return children[parentID][i].CreatedAt < children[parentID][j].CreatedAt
		})
	}

	replies := make([]SocialPost, 0)
	var walk func(string)
	walk = func(parentID string) {
		for _, child := range children[parentID] {
			if limit > 0 && len(replies) >= limit {
				return
			}
			replies = append(replies, child)
			walk(child.ID)
			if limit > 0 && len(replies) >= limit {
				return
			}
		}
	}

	walk(postID)
	return replies
}

func nextID(prefix string) string {
	return prefix + "_" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

func normalizeHandle(handle string) string {
	handle = strings.TrimSpace(handle)
	if !strings.HasPrefix(handle, "@") {
		return "@" + handle
	}
	return handle
}

func valueOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func slicePosts(posts []SocialPost, limit int) []SocialPost {
	if limit <= 0 || limit > len(posts) {
		limit = len(posts)
	}
	return posts[:limit]
}

func sliceTopLevelPosts(posts []SocialPost, limit int) []SocialPost {
	topLevel := make([]SocialPost, 0, len(posts))
	for _, post := range posts {
		if strings.TrimSpace(post.ParentPostID) == "" {
			topLevel = append(topLevel, post)
		}
	}
	if limit <= 0 || limit > len(topLevel) {
		limit = len(topLevel)
	}
	return topLevel[:limit]
}

func sliceMedia(media []MediaAsset, limit int) []MediaAsset {
	if limit <= 0 || limit > len(media) {
		limit = len(media)
	}
	return media[:limit]
}

func sliceConversations(conversations []Conversation, limit int) []Conversation {
	if limit <= 0 || limit > len(conversations) {
		limit = len(conversations)
	}
	return conversations[:limit]
}

package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
)

type UserField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type SocialUser struct {
	ID           string      `json:"id"`
	Handle       string      `json:"handle"`
	DisplayName  string      `json:"displayName"`
	Bio          string      `json:"bio"`
	Instance     string      `json:"instance"`
	Wallet       string      `json:"wallet"`
	AvatarURL    string      `json:"avatarUrl"`
	Fields       []UserField `json:"fields"`
	FeaturedTags []string    `json:"featuredTags"`
	IsBot        bool        `json:"isBot"`
	Followers    int         `json:"followers"`
	Following    int         `json:"following"`
	CreatedAt    string      `json:"createdAt"`
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

type PollOption struct {
	Label string `json:"label"`
	Votes int    `json:"votes"`
}

type Poll struct {
	Options   []PollOption `json:"options"`
	ExpiresAt string       `json:"expiresAt"`
	Multiple  bool         `json:"multiple"`
	Voters    []string     `json:"voters"`
}

type SocialPost struct {
	ID              string      `json:"id"`
	AuthorID        string      `json:"authorId"`
	AuthorHandle    string      `json:"authorHandle"`
	AuthorName      string      `json:"authorName"`
	Instance        string      `json:"instance"`
	Kind            string      `json:"kind"`
	Content         string      `json:"content"`
	Visibility      string      `json:"visibility"`
	StorageURI      string      `json:"storageUri"`
	AttestationURI  string      `json:"attestationUri"`
	ChainID         string      `json:"chainId,omitempty"`
	TxHash          string      `json:"txHash,omitempty"`
	ContractAddress string      `json:"contractAddress,omitempty"`
	ExplorerURL     string      `json:"explorerUrl,omitempty"`
	Tags            []string    `json:"tags"`
	Media           []PostMedia `json:"media"`
	ParentPostID    string      `json:"parentPostId,omitempty"`
	RootPostID      string      `json:"rootPostId,omitempty"`
	ReplyDepth      int         `json:"replyDepth,omitempty"`
	Replies         int         `json:"replies"`
	Boosts          int         `json:"boosts"`
	Likes           int         `json:"likes"`
	Type            string      `json:"type"`
	Interaction     string      `json:"interaction"`
	Poll            *Poll       `json:"poll,omitempty"`
	CreatedAt       string      `json:"createdAt"`
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
	InitiatorID    string        `json:"initiatorId,omitempty"`
	Encrypted      bool          `json:"encrypted"`
	Messages       []ChatMessage `json:"messages"`
	UpdatedAt      string        `json:"updatedAt"`
}

type PostAttestationResult struct {
	ChainID         string
	TxHash          string
	ContractAddress string
	ExplorerURL     string
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

type UpdateUserRequest struct {
	DisplayName  *string      `json:"displayName,omitempty"`
	Bio          *string      `json:"bio,omitempty"`
	AvatarURL    *string      `json:"avatarUrl,omitempty"`
	Fields       *[]UserField `json:"fields,omitempty"`
	FeaturedTags *[]string    `json:"featuredTags,omitempty"`
	IsBot        *bool        `json:"isBot,omitempty"`
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
	Type           string   `json:"type"`
	Interaction    string   `json:"interaction"`
	StorageURI     string   `json:"storageUri"`
	AttestationURI string   `json:"attestationUri"`
	Tags           []string `json:"tags"`
	MediaIDs       []string `json:"mediaIds"`
	ParentPostID   string   `json:"parentPostId"`
	RootPostID     string   `json:"rootPostId"`
	PollOptions    []string `json:"pollOptions"`
	PollExpiresIn  int      `json:"pollExpiresIn"` // in minutes
	PollMultiple   bool     `json:"pollMultiple"`
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
	follows       map[string]map[string]bool
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

	if shouldSeedSocialDefaults() {
		s.seedDefaults()
	} else {
		s.instances = nil
		s.users = nil
		s.media = nil
		s.posts = nil
		s.conversations = nil
	}
	s.persistLocked()
}

func shouldSeedSocialDefaults() bool {
	value := strings.TrimSpace(os.Getenv("SOCIAL_SEED"))
	if value != "" {
		value = strings.ToLower(value)
		return value == "1" || value == "true" || value == "yes" || value == "on"
	}

	if isProductionEnvironment() {
		return false
	}

	return true
}

func (s *SocialService) Reset(seed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if seed {
		s.seedDefaults()
	} else {
		s.instances = nil
		s.users = nil
		s.media = nil
		s.posts = nil
		s.conversations = nil
	}
	if s.rdb != nil {
		keys := []string{
			"social:snapshot:users",
			"social:snapshot:posts",
			"social:snapshot:media",
			"social:snapshot:conversations",
			"social:snapshot:instances",
		}
		_ = s.rdb.Del(s.ctx, keys...).Err()
	}
	s.persistLocked()
}

func (s *SocialService) loadSnapshotsFromRedis() bool {
	var users []SocialUser
	var posts []SocialPost
	var media []MediaAsset
	var conversations []Conversation
	var instances []FederationInstance
	var follows map[string]map[string]bool

	loaders := []struct {
		key    string
		target any
	}{
		{key: "social:snapshot:users", target: &users},
		{key: "social:snapshot:posts", target: &posts},
		{key: "social:snapshot:media", target: &media},
		{key: "social:snapshot:conversations", target: &conversations},
		{key: "social:snapshot:instances", target: &instances},
		{key: "social:snapshot:follows", target: &follows},
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
	if follows == nil {
		follows = map[string]map[string]bool{}
	}
	s.follows = follows
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
		{},
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
			InitiatorID:    "user_archive",
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

	s.follows = map[string]map[string]bool{
		"user_archive": {
			"user_librarian": true,
			"user_fedilab":   true,
		},
		"user_librarian": {
			"user_archive": true,
		},
		"user_fedilab": {
			"user_archive": true,
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
		"social:snapshot:follows":       s.follows,
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

	filteredConversations := sliceConversationsByParticipant(s.conversations, limit, currentUserID)
	payload := BootstrapPayload{
		Stats:         SocialStats{Users: len(s.users), Posts: len(s.posts), MediaAssets: len(s.media), Conversations: len(filteredConversations)},
		Feed:          append([]SocialPost(nil), sliceTopLevelPosts(s.posts, limit)...),
		Users:         append([]SocialUser(nil), s.users...),
		Media:         append([]MediaAsset(nil), sliceMedia(s.media, limit)...),
		Conversations: append([]Conversation(nil), filteredConversations...),
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

func (s *SocialService) BootstrapMine(limit int, currentUserID string) BootstrapPayload {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filteredConversations := sliceConversationsByParticipant(s.conversations, limit, currentUserID)
	payload := BootstrapPayload{
		Stats:         SocialStats{Users: len(s.users), Posts: len(s.posts), MediaAssets: len(s.media), Conversations: len(filteredConversations)},
		Feed:          append([]SocialPost(nil), sliceTopLevelPostsByAuthor(s.posts, limit, currentUserID)...),
		Users:         append([]SocialUser(nil), s.users...),
		Media:         append([]MediaAsset(nil), sliceMedia(s.media, limit)...),
		Conversations: append([]Conversation(nil), filteredConversations...),
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

func (s *SocialService) ListConversations(limit int, currentUserID string) []Conversation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Conversation(nil), sliceConversationsByParticipant(s.conversations, limit, currentUserID)...)
}

func sliceConversationsByParticipant(conversations []Conversation, limit int, currentUserID string) []Conversation {
	currentUserID = strings.TrimSpace(currentUserID)
	if currentUserID == "" {
		return sliceConversations(conversations, limit)
	}
	filtered := make([]Conversation, 0, len(conversations))
	for _, conversation := range conversations {
		for _, participantID := range conversation.ParticipantIDs {
			if participantID == currentUserID {
				filtered = append(filtered, conversation)
				break
			}
		}
	}
	return sliceConversations(filtered, limit)
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

func (s *SocialService) UpdateUser(id string, req UpdateUserRequest) (SocialUser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i, user := range s.users {
		if user.ID == id {
			idx = i
			break
		}
	}

	if idx == -1 {
		return SocialUser{}, errors.New("user not found: " + id)
	}

	user := &s.users[idx]
	if req.DisplayName != nil {
		user.DisplayName = strings.TrimSpace(*req.DisplayName)
	}
	if req.Bio != nil {
		user.Bio = strings.TrimSpace(*req.Bio)
	}
	if req.AvatarURL != nil {
		user.AvatarURL = strings.TrimSpace(*req.AvatarURL)
	}
	if req.Fields != nil {
		user.Fields = *req.Fields
	}
	if req.FeaturedTags != nil {
		user.FeaturedTags = *req.FeaturedTags
	}
	if req.IsBot != nil {
		user.IsBot = *req.IsBot
	}

	s.persistLocked()
	return *user, nil
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

func (s *SocialService) FeedMine(limit int, currentUserID string) []SocialPost {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]SocialPost(nil), sliceTopLevelPostsByAuthor(s.posts, limit, currentUserID)...)
}

func sliceTopLevelPostsByAuthor(posts []SocialPost, limit int, authorID string) []SocialPost {
	authorID = strings.TrimSpace(authorID)
	if authorID == "" {
		return nil
	}
	filtered := make([]SocialPost, 0, limit)
	for _, post := range posts {
		if strings.TrimSpace(post.ParentPostID) != "" {
			continue
		}
		if post.AuthorID != authorID {
			continue
		}
		filtered = append(filtered, post)
		if len(filtered) >= limit {
			break
		}
	}
	return filtered
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
		Kind:            kind,
		Content:         strings.TrimSpace(req.Content),
		Visibility:      valueOrDefault(strings.TrimSpace(req.Visibility), "public"),
		StorageURI:      storageURI,
		AttestationURI:  strings.TrimSpace(req.AttestationURI),
		ChainID:         "",
		TxHash:          "",
		ContractAddress: "",
		ExplorerURL:     "",
		Tags:            req.Tags,
		Media:           attachments,
		ParentPostID:   parentPostID,
		RootPostID:     rootPostID,
		ReplyDepth:     replyDepth,
		Type:           valueOrDefault(strings.TrimSpace(req.Type), "post"),
		Interaction:    valueOrDefault(strings.TrimSpace(req.Interaction), "anyone"),
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}

	if len(req.PollOptions) >= 2 {
		expiresIn := req.PollExpiresIn
		if expiresIn <= 0 {
			expiresIn = 1440 // 1 day default
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

	s.posts = append([]SocialPost{post}, s.posts...)
	s.refreshPostAggregatesLocked()
	s.sortLocked()
	s.persistLocked()
	return post, nil
}

func buildLocalStorageURI(req CreatePostRequest) string {
	// Deterministic digest for your own repository even before you wire external storage.
	canonical := map[string]any{
		"authorId":      strings.TrimSpace(req.AuthorID),
		"content":       strings.TrimSpace(req.Content),
		"tags":          req.Tags,
		"mediaIds":      req.MediaIDs,
		"parentPostId":  strings.TrimSpace(req.ParentPostID),
		"rootPostId":    strings.TrimSpace(req.RootPostID),
		"visibility":    strings.TrimSpace(req.Visibility),
		"type":          strings.TrimSpace(req.Type),
		"interaction":   strings.TrimSpace(req.Interaction),
		"pollOptions":   req.PollOptions,
		"pollExpiresIn": req.PollExpiresIn,
		"pollMultiple":  req.PollMultiple,
	}
	raw, _ := json.Marshal(canonical)
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("sha256://%x", sum[:])
}

func (s *SocialService) SetPostAttestation(postID string, result PostAttestationResult) (SocialPost, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, idx := s.findPostLocked(postID)
	if idx == -1 {
		return SocialPost{}, errors.New("post not found: " + postID)
	}
	s.posts[idx].AttestationURI = strings.TrimSpace(result.TxHash)
	s.posts[idx].ChainID = strings.TrimSpace(result.ChainID)
	s.posts[idx].TxHash = strings.TrimSpace(result.TxHash)
	s.posts[idx].ContractAddress = strings.TrimSpace(result.ContractAddress)
	s.posts[idx].ExplorerURL = strings.TrimSpace(result.ExplorerURL)
	s.persistLocked()
	return s.posts[idx], nil
}

func (s *SocialService) VotePoll(postID string, userID string, optionIndices []int) (SocialPost, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	post, idx := s.findPostLocked(postID)
	if idx == -1 {
		return SocialPost{}, errors.New("post not found")
	}

	if post.Poll == nil {
		return SocialPost{}, errors.New("post has no poll")
	}

	// Check expiration
	expiresAt, err := time.Parse(time.RFC3339, post.Poll.ExpiresAt)
	if err == nil && time.Now().UTC().After(expiresAt) {
		return SocialPost{}, errors.New("poll has expired")
	}

	// Double voting check
	for _, v := range post.Poll.Voters {
		if v == userID {
			return SocialPost{}, errors.New("already voted")
		}
	}

	// Validate indices
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

	// Record votes
	for _, optIdx := range optionIndices {
		post.Poll.Options[optIdx].Votes++
	}
	post.Poll.Voters = append(post.Poll.Voters, userID)

	s.posts[idx] = post
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

func (s *SocialService) CreateConversation(initiatorID string, req CreateConversationRequest) (Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(initiatorID) == "" {
		return Conversation{}, errors.New("initiator is required")
	}
	if _, err := s.findUserLocked(initiatorID); err != nil {
		return Conversation{}, err
	}

	for _, participantID := range req.ParticipantIDs {
		if _, err := s.findUserLocked(participantID); err != nil {
			return Conversation{}, err
		}
	}

	participantSet := map[string]bool{}
	for _, participantID := range req.ParticipantIDs {
		trimmed := strings.TrimSpace(participantID)
		if trimmed != "" {
			participantSet[trimmed] = true
		}
	}
	participantSet[initiatorID] = true

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

		participants := s.conversations[i].ParticipantIDs
		if len(participants) != 2 {
			return Conversation{}, errors.New("conversation must contain exactly two participants")
		}

		var peerID string
		if participants[0] == sender.ID {
			peerID = participants[1]
		} else {
			peerID = participants[0]
		}

		mutualFollow := s.isFollowingLocked(sender.ID, peerID) && s.isFollowingLocked(peerID, sender.ID)
		if !mutualFollow {
			if s.conversations[i].InitiatorID == "" {
				s.conversations[i].InitiatorID = sender.ID
			}
			if sender.ID != s.conversations[i].InitiatorID {
				return Conversation{}, errors.New("the other user has not followed you back yet")
			}
			if len(s.conversations[i].Messages) >= 1 {
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

func (s *SocialService) FollowUser(followerID, targetID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	followerID = strings.TrimSpace(followerID)
	targetID = strings.TrimSpace(targetID)
	if followerID == "" || targetID == "" {
		return errors.New("follow user id is required")
	}
	if followerID == targetID {
		return errors.New("cannot follow yourself")
	}
	if _, err := s.findUserLocked(followerID); err != nil {
		return err
	}
	if _, err := s.findUserLocked(targetID); err != nil {
		return err
	}

	if s.follows == nil {
		s.follows = map[string]map[string]bool{}
	}
	if s.follows[followerID] == nil {
		s.follows[followerID] = map[string]bool{}
	}
	s.follows[followerID][targetID] = true

	s.persistLocked()
	return nil
}

func (s *SocialService) UnfollowUser(followerID, targetID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.follows == nil {
		return nil
	}
	if s.follows[followerID] != nil {
		delete(s.follows[followerID], targetID)
		if len(s.follows[followerID]) == 0 {
			delete(s.follows, followerID)
		}
	}
	s.persistLocked()
	return nil
}

func (s *SocialService) isFollowingLocked(followerID, targetID string) bool {
	if s.follows == nil {
		return false
	}
	targets := s.follows[followerID]
	if targets == nil {
		return false
	}
	return targets[targetID]
}

func (a *App) bootstrapHandler(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 10)
	currentUser, _ := a.optionalAuthenticatedUser(r)
	currentUserID := ""
	if currentUser != nil {
		currentUserID = currentUser.ID
	}
	mineRaw := strings.TrimSpace(r.URL.Query().Get("mine"))
	mine := mineRaw == "1" || strings.EqualFold(mineRaw, "true") || strings.EqualFold(mineRaw, "yes") || strings.EqualFold(mineRaw, "on")
	payload, err := a.store.Social.Bootstrap(a.ctx, a.social, limit, currentUserID, mine)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": payload})
}

func (a *App) instancesHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": a.social.ListInstances()})
}

func (a *App) listUsersHandler(w http.ResponseWriter, _ *http.Request) {
	users, err := a.store.Social.ListUsers(a.ctx, a.social)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": users})
}

func (a *App) createUserHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid JSON"})
		return
	}
	user, err := a.store.Users.Create(a.ctx, req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "data": user})
}

func (a *App) getUserHandler(w http.ResponseWriter, r *http.Request) {
	user, err := a.store.Social.GetUser(a.ctx, a.social, mux.Vars(r)["id"])
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": user})
}

func (a *App) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	authUser, ok := a.requireAuthenticatedUser(w, r)
	if !ok {
		return
	}

	targetID := mux.Vars(r)["id"]
	if targetID != authUser.ID {
		writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "you can only update your own profile"})
		return
	}

	var req UpdateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid JSON"})
		return
	}

	user, err := a.store.Social.UpdateUser(a.ctx, a.social, targetID, req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": user})
}

func (a *App) feedHandler(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 20)
	mineRaw := strings.TrimSpace(r.URL.Query().Get("mine"))
	mine := mineRaw == "1" || strings.EqualFold(mineRaw, "true") || strings.EqualFold(mineRaw, "yes") || strings.EqualFold(mineRaw, "on")
	if mine {
		currentUser, _ := a.optionalAuthenticatedUser(r)
		currentUserID := ""
		if currentUser != nil {
			currentUserID = currentUser.ID
		}
		posts, err := a.store.Social.FeedMine(a.ctx, a.social, limit, currentUserID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": posts})
		return
	}
	posts, err := a.store.Social.Feed(a.ctx, a.social, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": posts})
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
	post, err := a.store.Social.CreatePost(a.ctx, a.social, req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	if strings.TrimSpace(post.AttestationURI) == "" {
		txHash, attestErr := a.executePostAttestation(post.StorageURI)
		if attestErr == nil {
			chainPart := "0"
			if a.chainID != nil {
				chainPart = a.chainID.String()
			}
			explorerBase := strings.TrimSpace(os.Getenv("POST_ATTEST_EXPLORER_BASE_URL"))
			explorerURL := ""
			if explorerBase != "" {
				explorerURL = strings.TrimRight(explorerBase, "/") + "/tx/" + txHash
			}
			updated, setErr := a.social.SetPostAttestation(post.ID, PostAttestationResult{
				ChainID:         chainPart,
				TxHash:          txHash,
				ContractAddress: strings.TrimSpace(os.Getenv("POST_ATTEST_CONTRACT")),
				ExplorerURL:     explorerURL,
			})
			if setErr == nil {
				post = updated
			}
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "data": post})
}

type VotePollRequest struct {
	OptionIndices []int `json:"optionIndices"`
}

func (a *App) votePollHandler(w http.ResponseWriter, r *http.Request) {
	authUser, ok := a.requireAuthenticatedUser(w, r)
	if !ok {
		return
	}

	postID := mux.Vars(r)["id"]
	var req VotePollRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid JSON"})
		return
	}

	post, err := a.store.Social.VotePoll(a.ctx, a.social, postID, authUser.ID, req.OptionIndices)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": post})
}

func (a *App) getPostHandler(w http.ResponseWriter, r *http.Request) {
	post, err := a.store.Social.GetPost(a.ctx, a.social, mux.Vars(r)["id"])
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": post})
}

func (a *App) getPostThreadHandler(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 20)
	thread, err := a.store.Social.GetPostThread(a.ctx, a.social, mux.Vars(r)["id"], limit)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": thread})
}

func (a *App) listPostRepliesHandler(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 20)
	replies, err := a.store.Social.ListReplies(a.ctx, a.social, mux.Vars(r)["id"], limit)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": replies})
}

func (a *App) listMediaHandler(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 20)
	media, err := a.store.Social.ListMedia(a.ctx, a.social, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": media})
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
	asset, err := a.store.Social.CreateMedia(a.ctx, a.social, req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "data": asset})
}

func (a *App) listConversationsHandler(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 20)
	currentUser, _ := a.optionalAuthenticatedUser(r)
	currentUserID := ""
	if currentUser != nil {
		currentUserID = currentUser.ID
	}
	conversations, err := a.store.Social.ListConversations(a.ctx, a.social, limit, currentUserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": conversations})
}

func (a *App) createConversationHandler(w http.ResponseWriter, r *http.Request) {
	authUser, ok := a.requireAuthenticatedUser(w, r)
	if !ok {
		return
	}

	var req CreateConversationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid JSON"})
		return
	}
	conversation, err := a.store.Social.CreateConversation(a.ctx, a.social, authUser.ID, req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "data": conversation})
}

func (a *App) followUserHandler(w http.ResponseWriter, r *http.Request) {
	authUser, ok := a.requireAuthenticatedUser(w, r)
	if !ok {
		return
	}

	targetID := mux.Vars(r)["id"]
	if err := a.store.Social.FollowUser(a.ctx, a.social, authUser.ID, targetID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) unfollowUserHandler(w http.ResponseWriter, r *http.Request) {
	authUser, ok := a.requireAuthenticatedUser(w, r)
	if !ok {
		return
	}

	targetID := mux.Vars(r)["id"]
	if err := a.store.Social.UnfollowUser(a.ctx, authUser.ID, targetID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) getConversationHandler(w http.ResponseWriter, r *http.Request) {
	conversation, err := a.store.Social.GetConversation(a.ctx, a.social, mux.Vars(r)["id"])
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
	conversation, err := a.store.Social.AddMessage(a.ctx, a.social, mux.Vars(r)["id"], req)
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

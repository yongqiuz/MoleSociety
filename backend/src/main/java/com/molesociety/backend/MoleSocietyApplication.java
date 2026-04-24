package com.molesociety.backend;

import jakarta.servlet.http.Cookie;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.io.IOException;
import java.net.HttpURLConnection;
import java.math.BigInteger;
import java.net.URI;
import java.net.URLDecoder;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.charset.StandardCharsets;
import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Statement;
import java.sql.Timestamp;
import java.sql.Types;
import java.security.MessageDigest;
import java.security.SecureRandom;
import java.time.Duration;
import java.time.Instant;
import java.time.format.DateTimeParseException;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.HashMap;
import java.util.HashSet;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Objects;
import java.util.Optional;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import org.bouncycastle.jcajce.provider.digest.Keccak;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.context.annotation.Bean;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseCookie;
import org.springframework.http.ResponseEntity;
import org.springframework.security.crypto.bcrypt.BCryptPasswordEncoder;
import org.springframework.stereotype.Service;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PatchMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.servlet.config.annotation.CorsRegistry;
import org.springframework.web.servlet.config.annotation.WebMvcConfigurer;
import redis.clients.jedis.JedisPooled;

@SpringBootApplication
public class MoleSocietyApplication {
  public static void main(String[] args) {
    loadDotEnv();
    SpringApplication.run(MoleSocietyApplication.class, args);
  }

  private static void loadDotEnv() {
    Path env = Path.of(".env");
    if (!Files.exists(env)) {
      env = Path.of("backend", ".env");
    }
    if (!Files.exists(env)) {
      return;
    }
    try {
      for (String line : Files.readAllLines(env)) {
        String trimmed = line.trim();
        if (trimmed.isEmpty() || trimmed.startsWith("#") || !trimmed.contains("=")) {
          continue;
        }
        int idx = trimmed.indexOf('=');
        String key = trimmed.substring(0, idx).trim();
        String value = trimmed.substring(idx + 1).trim();
        if (!key.isEmpty() && System.getProperty(key) == null && System.getenv(key) == null) {
          System.setProperty(key, value);
        }
      }
    } catch (IOException ignored) {
    }
  }

  @Bean
  WebMvcConfigurer corsConfigurer() {
    return new WebMvcConfigurer() {
      @Override
      public void addCorsMappings(CorsRegistry registry) {
        registry.addMapping("/**")
            .allowedOriginPatterns(
                "http://localhost:*",
                "https://localhost:*",
                "http://127.0.0.1:*",
                "https://127.0.0.1:*",
                "http://192.168.*:*",
                "https://192.168.*:*",
                "http://10.*:*",
                "https://10.*:*",
                "http://172.*:*",
                "https://172.*:*")
            .allowedMethods("GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS")
            .allowedHeaders("Content-Type", "Authorization", "X-Admin-Wallet")
            .allowCredentials(true);
      }
    };
  }
}

@RestController
class HealthController {
  private final SocialService social;
  private final PersistenceService persistence;

  HealthController(SocialService social, PersistenceService persistence) {
    this.social = social;
    this.persistence = persistence;
  }

  @GetMapping("/healthz")
  Map<String, Object> health() {
    return Map.of(
        "ok", true,
        "service", "molesociety-springboot-backend",
        "databaseMode", persistence.databaseMode(),
        "database", persistence.databaseHealth(),
        "migrations", persistence.migrationsStatus(),
        "redis", persistence.redisAvailable(),
        "relayReady", false,
        "socialSeed", Env.truthy(Env.get("SOCIAL_SEED", persistence.databaseAvailable() ? "0" : "1")),
        "appEnv", Env.current(),
        "socialStats", social.stats());
  }
}

@RestController
@RequestMapping("/api/v1/auth")
class AuthController {
  private final AuthService auth;

  AuthController(AuthService auth) {
    this.auth = auth;
  }

  @PostMapping("/challenge")
  ResponseEntity<ApiResponse<AuthChallenge>> challenge(@RequestBody AuthChallengeRequest req, HttpServletRequest request) {
    return ApiResponse.ok(auth.createChallenge(req.address, req.chainId, requestOrigin(request)));
  }

  @PostMapping("/bind-challenge")
  ResponseEntity<ApiResponse<AuthChallenge>> bindChallenge(@RequestBody BindChallengeRequest req, HttpServletRequest request) {
    return ApiResponse.ok(auth.createChallenge(req.walletAddress, req.chainId, requestOrigin(request)));
  }

  @PostMapping("/verify")
  ResponseEntity<ApiResponse<AuthSessionResponse>> verify(@RequestBody VerifyWalletRequest req, HttpServletResponse response) {
    try {
      AuthResult result = auth.verifyWalletLogin(req.address, req.nonce, req.signature);
      setSessionCookie(response, result.session.id);
      return ApiResponse.ok(AuthSessionResponse.from(result.user));
    } catch (ApiException err) {
      return err.toResponse();
    }
  }

  @PostMapping("/password-login")
  ResponseEntity<ApiResponse<AuthSessionResponse>> passwordLogin(@RequestBody PasswordLoginRequest req, HttpServletResponse response) {
    try {
      AuthResult result = auth.passwordLogin(req.identifier, req.password);
      setSessionCookie(response, result.session.id);
      return ApiResponse.ok(AuthSessionResponse.from(result.user));
    } catch (ApiException err) {
      return err.toResponse();
    }
  }

  @PostMapping("/register")
  ResponseEntity<ApiResponse<AuthSessionResponse>> register(@RequestBody RegisterRequest req, HttpServletRequest request, HttpServletResponse response) {
    try {
      AuthResult result = auth.register(req, requestOrigin(request));
      setSessionCookie(response, result.session.id);
      return ApiResponse.created(AuthSessionResponse.from(result.user));
    } catch (ApiException err) {
      return err.toResponse();
    }
  }

  @GetMapping("/me")
  ResponseEntity<ApiResponse<AuthSessionResponse>> me(HttpServletRequest request, HttpServletResponse response) {
    try {
      SocialUser user = auth.userFromRequest(request).orElseThrow(() -> ApiException.unauthorized("AUTH_SESSION_REQUIRED", "session", "authentication required"));
      return ApiResponse.ok(AuthSessionResponse.from(user));
    } catch (ApiException err) {
      clearSessionCookie(response);
      return err.toResponse();
    }
  }

  @PostMapping("/logout")
  ResponseEntity<ApiResponse<Map<String, Boolean>>> logout(HttpServletRequest request, HttpServletResponse response) {
    sessionCookie(request).ifPresent(auth::deleteSession);
    clearSessionCookie(response);
    return ApiResponse.ok(Map.of("loggedOut", true));
  }

  private String requestOrigin(HttpServletRequest request) {
    String origin = request.getHeader("Origin");
    if (Strings.hasText(origin)) {
      return origin;
    }
    return request.getScheme() + "://" + request.getServerName() + ":" + request.getServerPort();
  }

  private static void setSessionCookie(HttpServletResponse response, String sessionID) {
    ResponseCookie cookie = ResponseCookie.from(AuthService.SESSION_COOKIE, sessionID)
        .httpOnly(true)
        .secure(Env.cookieSecure())
        .sameSite(Env.cookieSameSite())
        .path("/")
        .maxAge(Duration.ofDays(7))
        .build();
    response.addHeader(HttpHeaders.SET_COOKIE, cookie.toString());
  }

  private static void clearSessionCookie(HttpServletResponse response) {
    ResponseCookie cookie = ResponseCookie.from(AuthService.SESSION_COOKIE, "")
        .httpOnly(true)
        .secure(Env.cookieSecure())
        .sameSite(Env.cookieSameSite())
        .path("/")
        .maxAge(Duration.ZERO)
        .build();
    response.addHeader(HttpHeaders.SET_COOKIE, cookie.toString());
  }

  static Optional<String> sessionCookie(HttpServletRequest request) {
    Cookie[] cookies = request.getCookies();
    if (cookies == null) {
      return Optional.empty();
    }
    for (Cookie cookie : cookies) {
      if (AuthService.SESSION_COOKIE.equals(cookie.getName()) && Strings.hasText(cookie.getValue())) {
        return Optional.of(cookie.getValue().trim());
      }
    }
    return Optional.empty();
  }
}

@RestController
@RequestMapping("/api/v1/social")
class SocialController {
  private final SocialService social;
  private final AuthService auth;

  SocialController(SocialService social, AuthService auth) {
    this.social = social;
    this.auth = auth;
  }

  @GetMapping("/bootstrap")
  ResponseEntity<ApiResponse<BootstrapPayload>> bootstrap(@RequestParam(defaultValue = "20") int limit, @RequestParam(defaultValue = "0") String mine, HttpServletRequest request) {
    String userID = auth.userFromRequest(request).map(u -> u.id).orElse("");
    return ApiResponse.ok(social.bootstrap(limit, userID, Env.truthy(mine)));
  }

  @GetMapping("/instances")
  ResponseEntity<ApiResponse<List<FederationInstance>>> instances() {
    return ApiResponse.ok(social.listInstances());
  }

  @GetMapping("/users")
  ResponseEntity<ApiResponse<List<SocialUser>>> users() {
    return ApiResponse.ok(social.listUsers());
  }

  @PostMapping("/users")
  ResponseEntity<ApiResponse<SocialUser>> createUser(@RequestBody CreateUserRequest req) {
    try {
      return ApiResponse.created(social.createUser(req));
    } catch (ApiException err) {
      return err.toResponse();
    }
  }

  @GetMapping("/users/{id}")
  ResponseEntity<ApiResponse<SocialUser>> user(@PathVariable String id) {
    try {
      return ApiResponse.ok(social.getUser(id));
    } catch (ApiException err) {
      return err.toResponse();
    }
  }

  @PatchMapping("/users/{id}")
  ResponseEntity<ApiResponse<SocialUser>> updateUser(@PathVariable String id, @RequestBody UpdateUserRequest req, HttpServletRequest request) {
    try {
      SocialUser current = requiredUser(request);
      if (!current.id.equals(id)) {
        throw ApiException.forbidden("AUTH_FORBIDDEN", "authorization", "you can only update your own profile");
      }
      return ApiResponse.ok(social.updateUser(id, req));
    } catch (ApiException err) {
      return err.toResponse();
    }
  }

  @PostMapping("/users/{id}/follow")
  ResponseEntity<ApiResponse<Map<String, Boolean>>> follow(@PathVariable String id, HttpServletRequest request) {
    try {
      social.followUser(requiredUser(request).id, id);
      return ApiResponse.ok(Map.of("followed", true));
    } catch (ApiException err) {
      return err.toResponse();
    }
  }

  @DeleteMapping("/users/{id}/follow")
  ResponseEntity<ApiResponse<Map<String, Boolean>>> unfollow(@PathVariable String id, HttpServletRequest request) {
    try {
      social.unfollowUser(requiredUser(request).id, id);
      return ApiResponse.ok(Map.of("unfollowed", true));
    } catch (ApiException err) {
      return err.toResponse();
    }
  }

  @GetMapping("/feed")
  ResponseEntity<ApiResponse<List<SocialPost>>> feed(@RequestParam(defaultValue = "20") int limit, @RequestParam(defaultValue = "0") String mine, HttpServletRequest request) {
    String userID = auth.userFromRequest(request).map(u -> u.id).orElse("");
    return ApiResponse.ok(Env.truthy(mine) ? social.feedMine(limit, userID) : social.feed(limit));
  }

  @PostMapping("/posts")
  ResponseEntity<ApiResponse<SocialPost>> createPost(@RequestBody CreatePostRequest req, HttpServletRequest request) {
    try {
      req.authorId = requiredUser(request).id;
      return ApiResponse.created(social.createPost(req));
    } catch (ApiException err) {
      return err.toResponse();
    }
  }

  @GetMapping("/posts/{id}")
  ResponseEntity<ApiResponse<SocialPost>> post(@PathVariable String id) {
    try {
      return ApiResponse.ok(social.getPost(id));
    } catch (ApiException err) {
      return err.toResponse();
    }
  }

  @GetMapping("/posts/{id}/thread")
  ResponseEntity<ApiResponse<PostThread>> thread(@PathVariable String id, @RequestParam(defaultValue = "20") int limit) {
    try {
      return ApiResponse.ok(social.getPostThread(id, limit));
    } catch (ApiException err) {
      return err.toResponse();
    }
  }

  @GetMapping("/posts/{id}/replies")
  ResponseEntity<ApiResponse<List<SocialPost>>> replies(@PathVariable String id, @RequestParam(defaultValue = "20") int limit) {
    try {
      return ApiResponse.ok(social.listReplies(id, limit));
    } catch (ApiException err) {
      return err.toResponse();
    }
  }

  @PostMapping("/posts/{id}/poll/vote")
  ResponseEntity<ApiResponse<SocialPost>> vote(@PathVariable String id, @RequestBody VotePollRequest req, HttpServletRequest request) {
    try {
      return ApiResponse.ok(social.votePoll(id, requiredUser(request).id, req.optionIndices));
    } catch (ApiException err) {
      return err.toResponse();
    }
  }

  @GetMapping("/media")
  ResponseEntity<ApiResponse<List<MediaAsset>>> media(@RequestParam(defaultValue = "20") int limit) {
    return ApiResponse.ok(social.listMedia(limit));
  }

  @PostMapping("/media")
  ResponseEntity<ApiResponse<MediaAsset>> createMedia(@RequestBody CreateMediaRequest req, HttpServletRequest request) {
    try {
      req.ownerId = requiredUser(request).id;
      return ApiResponse.created(social.createMedia(req));
    } catch (ApiException err) {
      return err.toResponse();
    }
  }

  @GetMapping("/conversations")
  ResponseEntity<ApiResponse<List<Conversation>>> conversations(@RequestParam(defaultValue = "20") int limit, HttpServletRequest request) {
    String userID = auth.userFromRequest(request).map(u -> u.id).orElse("");
    return ApiResponse.ok(social.listConversations(limit, userID));
  }

  @PostMapping("/conversations")
  ResponseEntity<ApiResponse<Conversation>> createConversation(@RequestBody CreateConversationRequest req, HttpServletRequest request) {
    try {
      return ApiResponse.created(social.createConversation(requiredUser(request).id, req));
    } catch (ApiException err) {
      return err.toResponse();
    }
  }

  @GetMapping("/conversations/{id}")
  ResponseEntity<ApiResponse<Conversation>> conversation(@PathVariable String id) {
    try {
      return ApiResponse.ok(social.getConversation(id));
    } catch (ApiException err) {
      return err.toResponse();
    }
  }

  @PostMapping("/conversations/{id}/messages")
  ResponseEntity<ApiResponse<Conversation>> message(@PathVariable String id, @RequestBody CreateMessageRequest req, HttpServletRequest request) {
    try {
      req.senderId = requiredUser(request).id;
      return ApiResponse.created(social.addMessage(id, req));
    } catch (ApiException err) {
      return err.toResponse();
    }
  }

  private SocialUser requiredUser(HttpServletRequest request) {
    return auth.userFromRequest(request).orElseThrow(() -> ApiException.unauthorized("AUTH_SESSION_REQUIRED", "session", "authentication required"));
  }
}

@RestController
class LegacyController {
  private final SocialService social;

  LegacyController(SocialService social) {
    this.social = social;
  }

  @GetMapping("/api/admin/check-access")
  Map<String, Object> checkAccess(@RequestParam(defaultValue = "") String address, @RequestHeader(value = "X-Admin-Wallet", required = false) String headerAddress) {
    String resolved = Strings.hasText(address) ? address : Strings.value(headerAddress);
    return Map.of("ok", true, "address", resolved, "access", Strings.hasText(resolved));
  }

  @PostMapping("/api/admin/social/reset")
  ResponseEntity<ApiResponse<Map<String, Object>>> reset(@RequestParam(defaultValue = "1") String seed) {
    social.reset(Env.truthy(seed));
    return ApiResponse.ok(Map.of("seed", Env.truthy(seed), "socialStats", social.stats()));
  }

  @GetMapping("/api/v1/analytics/distribution")
  Map<String, Object> distribution() {
    return Map.of("ok", true, "distribution", social.distribution());
  }

  @GetMapping("/secret/get-binding")
  Map<String, Object> binding() {
    return Map.of("ok", true);
  }

  @GetMapping("/secret/verify")
  Map<String, Object> verify() {
    return Map.of("ok", true, "role", "reader");
  }

  @PostMapping("/relay/mint")
  Map<String, Object> mint() {
    return Map.of("ok", true, "role", "reader", "txHash", "simulated-" + System.nanoTime());
  }

  @PostMapping("/relay/save-code")
  Map<String, Object> saveCode() {
    return Map.of("ok", true, "status", "saved");
  }

  @PostMapping("/relay/reward")
  Map<String, Object> reward(@RequestBody(required = false) Map<String, Object> body) {
    Map<String, Object> result = new LinkedHashMap<>();
    result.put("ok", true);
    result.put("status", "queued");
    if (body != null) {
      result.putAll(body);
    }
    return result;
  }

  @GetMapping("/relay/stats")
  Map<String, Object> stats() {
    return Map.of("ok", true, "referrers", List.of(), "socialStats", social.stats());
  }
}

@Service
class PersistenceService {
  private static final ObjectMapper JSON = new ObjectMapper();
  private final String databaseUrl;
  private final String migrationsDir;
  private final List<String> migrationFiles = new ArrayList<>();
  private final List<String> appliedMigrations = new ArrayList<>();
  private boolean databaseAvailable;
  private JedisPooled redis;

  PersistenceService() {
    this.databaseUrl = Strings.value(Env.get("DATABASE_URL", "")).trim();
    this.migrationsDir = Strings.or(Env.get("DB_MIGRATIONS_DIR", ""), "./migrations");
    initDatabase();
    initRedis();
  }

  boolean databaseAvailable() {
    return databaseAvailable;
  }

  boolean redisAvailable() {
    return redis != null;
  }

  String databaseMode() {
    if (databaseAvailable && redisAvailable()) {
      return "postgres(jdbc)+redis";
    }
    if (databaseAvailable) {
      return "postgres(jdbc)";
    }
    if (redisAvailable()) {
      return "memory+redis";
    }
    return "memory";
  }

  Map<String, Object> databaseHealth() {
    Map<String, Object> payload = new LinkedHashMap<>();
    payload.put("enabled", databaseAvailable);
    payload.put("driver", "jdbc-postgresql");
    payload.put("migrationsDir", migrationsDir);
    payload.put("mode", databaseAvailable ? "connected" : "disabled");
    return payload;
  }

  Map<String, Object> migrationsStatus() {
    return Map.of("count", migrationFiles.size(), "files", migrationFiles, "applied", appliedMigrations);
  }

  boolean loadSocialState(SocialService social) {
    if (databaseAvailable) {
      SocialState state = loadSocialFromPostgres();
      if (!state.users.isEmpty() || !state.posts.isEmpty() || !state.instances.isEmpty()) {
        social.replaceState(state);
        saveSocialSnapshot(state);
        return true;
      }
    }
    Optional<SocialState> redisState = loadSocialSnapshot();
    redisState.ifPresent(social::replaceState);
    return redisState.isPresent();
  }

  void replaceSocialState(SocialState state) {
    if (databaseAvailable) {
      exec(conn -> {
        try (Statement st = conn.createStatement()) {
          st.executeUpdate("DELETE FROM chat_messages");
          st.executeUpdate("DELETE FROM conversation_participants");
          st.executeUpdate("DELETE FROM conversations");
          st.executeUpdate("DELETE FROM post_media_links");
          st.executeUpdate("DELETE FROM social_posts");
          st.executeUpdate("DELETE FROM media_assets");
          st.executeUpdate("DELETE FROM user_follows");
          st.executeUpdate("DELETE FROM auth_accounts");
          st.executeUpdate("DELETE FROM social_users");
          st.executeUpdate("DELETE FROM federation_instances");
        }
      });
      for (FederationInstance item : state.instances) saveInstance(item);
      for (SocialUser user : state.users) saveUser(user);
      for (MediaAsset asset : state.media) saveMedia(asset);
      for (SocialPost post : postsForPersistence(state.posts)) savePost(post);
      for (Conversation conversation : state.conversations) saveConversation(conversation);
      for (Map.Entry<String, Set<String>> entry : state.follows.entrySet()) {
        for (String target : entry.getValue()) saveFollow(entry.getKey(), target);
      }
    }
    saveSocialSnapshot(state);
  }

  void saveLoadedSocialState(SocialState state) {
    if (databaseAvailable) {
      exec(conn -> {
        try (Statement st = conn.createStatement()) {
          st.executeUpdate("DELETE FROM federation_instances");
        }
      });
      for (FederationInstance item : state.instances) saveInstance(item);
      for (SocialUser user : state.users) saveUser(user);
      for (MediaAsset asset : state.media) saveMedia(asset);
      for (SocialPost post : postsForPersistence(state.posts)) savePost(post);
      for (Conversation conversation : state.conversations) saveConversation(conversation);
      for (Map.Entry<String, Set<String>> entry : state.follows.entrySet()) {
        for (String target : entry.getValue()) saveFollow(entry.getKey(), target);
      }
    }
    saveSocialSnapshot(state);
  }

  private List<SocialPost> postsForPersistence(List<SocialPost> posts) {
    Set<String> postIDs = new HashSet<>();
    for (SocialPost post : posts) postIDs.add(post.id);
    for (SocialPost post : posts) {
      if (Strings.hasText(post.parentPostId) && !postIDs.contains(post.parentPostId)) {
        post.parentPostId = "";
        post.replyDepth = 0;
      }
      if (Strings.hasText(post.rootPostId) && !postIDs.contains(post.rootPostId)) {
        post.rootPostId = "";
      }
    }
    return posts.stream()
        .sorted(Comparator
            .comparingInt((SocialPost post) -> Strings.hasText(post.parentPostId) ? 1 : 0)
            .thenComparing(post -> Strings.value(post.createdAt)))
        .toList();
  }

  void saveSocialSnapshot(SocialState state) {
    if (redis == null) return;
    setJson("social:snapshot:users", state.users, 0);
    setJson("social:snapshot:posts", state.posts, 0);
    setJson("social:snapshot:media", state.media, 0);
    setJson("social:snapshot:conversations", state.conversations, 0);
    setJson("social:snapshot:instances", state.instances, 0);
    setJson("social:snapshot:follows", state.follows, 0);
  }

  Optional<SocialState> loadSocialSnapshot() {
    if (redis == null) return Optional.empty();
    try {
      SocialState state = new SocialState();
      state.users.addAll(readJson("social:snapshot:users", new TypeReference<List<SocialUser>>() {}));
      state.posts.addAll(readJson("social:snapshot:posts", new TypeReference<List<SocialPost>>() {}));
      state.media.addAll(readJson("social:snapshot:media", new TypeReference<List<MediaAsset>>() {}));
      state.conversations.addAll(readJson("social:snapshot:conversations", new TypeReference<List<Conversation>>() {}));
      state.instances.addAll(readJson("social:snapshot:instances", new TypeReference<List<FederationInstance>>() {}));
      state.follows.putAll(readJson("social:snapshot:follows", new TypeReference<Map<String, Set<String>>>() {}));
      return Optional.of(state);
    } catch (RuntimeException err) {
      return Optional.empty();
    }
  }

  void saveChallenge(AuthChallenge challenge, Duration ttl) {
    if (redis != null) setJson("auth:challenge:" + challenge.nonce, challenge, (int) ttl.toSeconds());
  }

  Optional<AuthChallenge> loadChallenge(String nonce) {
    if (redis == null) return Optional.empty();
    try {
      String raw = redis.get("auth:challenge:" + nonce);
      if (!Strings.hasText(raw)) return Optional.empty();
      return Optional.of(JSON.readValue(raw, AuthChallenge.class));
    } catch (Exception err) {
      return Optional.empty();
    }
  }

  void deleteChallenge(String nonce) {
    if (redis != null && Strings.hasText(nonce)) redis.del("auth:challenge:" + nonce);
  }

  void saveSession(AuthSessionRecord session, Duration ttl) {
    if (redis != null) setJson("auth:session:" + session.id, session, (int) ttl.toSeconds());
  }

  Optional<AuthSessionRecord> loadSession(String id) {
    if (redis == null) return Optional.empty();
    try {
      String raw = redis.get("auth:session:" + id);
      if (!Strings.hasText(raw)) return Optional.empty();
      return Optional.of(JSON.readValue(raw, AuthSessionRecord.class));
    } catch (Exception err) {
      return Optional.empty();
    }
  }

  void deleteSession(String id) {
    if (redis != null && Strings.hasText(id)) redis.del("auth:session:" + id);
  }

  Map<String, Account> loadAuthAccounts() {
    Map<String, Account> result = new ConcurrentHashMap<>();
    if (databaseAvailable) {
      query(conn -> {
        try (PreparedStatement ps = conn.prepareStatement("SELECT id, username, email, password_hash, wallet, user_id, status, created_at, updated_at FROM auth_accounts")) {
          ResultSet rs = ps.executeQuery();
          while (rs.next()) {
            Account account = new Account();
            account.id = rs.getString("id");
            account.username = rs.getString("username");
            account.email = Strings.value(rs.getString("email"));
            account.passwordHash = rs.getString("password_hash");
            account.wallet = Strings.value(rs.getString("wallet"));
            account.userId = rs.getString("user_id");
            account.status = rs.getString("status");
            account.createdAt = instantString(rs.getTimestamp("created_at"));
            account.updatedAt = instantString(rs.getTimestamp("updated_at"));
            result.put(account.id, account);
          }
        }
      });
      if (!result.isEmpty()) return result;
    }
    if (redis != null) {
      try {
        String raw = redis.get("auth:accounts:snapshot");
        if (Strings.hasText(raw)) {
          List<Account> accounts = JSON.readValue(raw, new TypeReference<List<Account>>() {});
          for (Account account : accounts) result.put(account.id, account);
        }
      } catch (Exception ignored) {
      }
    }
    return result;
  }

  void saveAuthAccount(Account account) {
    if (databaseAvailable) {
      exec(conn -> {
        try (PreparedStatement ps = conn.prepareStatement("""
            INSERT INTO auth_accounts(id, username, email, password_hash, wallet, user_id, status, created_at, updated_at)
            VALUES (?,?,?,?,?,?,?,?,?)
            ON CONFLICT (id) DO UPDATE SET username=EXCLUDED.username, email=EXCLUDED.email,
              password_hash=EXCLUDED.password_hash, wallet=EXCLUDED.wallet, user_id=EXCLUDED.user_id,
              status=EXCLUDED.status, updated_at=EXCLUDED.updated_at
            """)) {
          ps.setString(1, account.id);
          ps.setString(2, account.username);
          setNullable(ps, 3, account.email);
          ps.setString(4, account.passwordHash);
          setNullable(ps, 5, account.wallet);
          ps.setString(6, account.userId);
          ps.setString(7, Strings.or(account.status, "active"));
          ps.setTimestamp(8, timestamp(account.createdAt));
          ps.setTimestamp(9, timestamp(account.updatedAt));
          ps.executeUpdate();
        }
      });
    }
    if (redis != null) {
      Map<String, Account> accounts = loadAuthAccounts();
      accounts.put(account.id, account);
      setJson("auth:accounts:snapshot", new ArrayList<>(accounts.values()), 0);
    }
  }

  void saveInstance(FederationInstance item) {
    if (!databaseAvailable) return;
    exec(conn -> {
      try (PreparedStatement ps = conn.prepareStatement("""
          INSERT INTO federation_instances(name, focus, members, latency, status)
          VALUES (?,?,?,?,?)
          ON CONFLICT (name) DO UPDATE SET focus=EXCLUDED.focus, members=EXCLUDED.members,
            latency=EXCLUDED.latency, status=EXCLUDED.status
          """)) {
        ps.setString(1, item.name);
        ps.setString(2, item.focus);
        ps.setString(3, item.members);
        ps.setString(4, item.latency);
        ps.setString(5, item.status);
        ps.executeUpdate();
      }
    });
  }

  void saveUser(SocialUser user) {
    if (!databaseAvailable) return;
    exec(conn -> {
      try (PreparedStatement ps = conn.prepareStatement("""
          INSERT INTO social_users(id, handle, display_name, bio, instance, wallet, avatar_url,
            fields_json, featured_tags_json, is_bot, followers_count, following_count, created_at)
          VALUES (?,?,?,?,?,?,?,?::jsonb,?::jsonb,?,?,?,?)
          ON CONFLICT (id) DO UPDATE SET handle=EXCLUDED.handle, display_name=EXCLUDED.display_name,
            bio=EXCLUDED.bio, instance=EXCLUDED.instance, wallet=EXCLUDED.wallet, avatar_url=EXCLUDED.avatar_url,
            fields_json=EXCLUDED.fields_json, featured_tags_json=EXCLUDED.featured_tags_json,
            is_bot=EXCLUDED.is_bot, followers_count=EXCLUDED.followers_count, following_count=EXCLUDED.following_count
          """)) {
        ps.setString(1, user.id);
        ps.setString(2, user.handle);
        ps.setString(3, user.displayName);
        ps.setString(4, Strings.value(user.bio));
        ps.setString(5, user.instance);
        ps.setString(6, Strings.value(user.wallet));
        ps.setString(7, Strings.value(user.avatarUrl));
        ps.setString(8, json(user.fields));
        ps.setString(9, json(user.featuredTags));
        ps.setBoolean(10, user.isBot);
        ps.setInt(11, user.followers);
        ps.setInt(12, user.following);
        ps.setTimestamp(13, timestamp(user.createdAt));
        ps.executeUpdate();
      }
    });
  }

  void saveMedia(MediaAsset asset) {
    if (!databaseAvailable) return;
    exec(conn -> {
      try (PreparedStatement ps = conn.prepareStatement("""
          INSERT INTO media_assets(id, owner_id, name, kind, url, storage_uri, cid, size_bytes, status, created_at)
          VALUES (?,?,?,?,?,?,?,?,?,?)
          ON CONFLICT (id) DO UPDATE SET owner_id=EXCLUDED.owner_id, name=EXCLUDED.name, kind=EXCLUDED.kind,
            url=EXCLUDED.url, storage_uri=EXCLUDED.storage_uri, cid=EXCLUDED.cid, size_bytes=EXCLUDED.size_bytes,
            status=EXCLUDED.status
          """)) {
        ps.setString(1, asset.id);
        ps.setString(2, asset.ownerId);
        ps.setString(3, asset.name);
        ps.setString(4, asset.kind);
        ps.setString(5, Strings.value(asset.url));
        ps.setString(6, Strings.value(asset.storageUri));
        ps.setString(7, Strings.value(asset.cid));
        ps.setLong(8, asset.sizeBytes);
        ps.setString(9, asset.status);
        ps.setTimestamp(10, timestamp(asset.createdAt));
        ps.executeUpdate();
      }
    });
  }

  void savePost(SocialPost post) {
    if (!databaseAvailable) return;
    exec(conn -> {
      try (PreparedStatement ps = conn.prepareStatement("""
          INSERT INTO social_posts(id, author_id, instance, kind, content, visibility, storage_uri, attestation_uri,
            chain_id, tx_hash, contract_address, explorer_url, tags_json, parent_post_id, root_post_id,
            reply_depth, replies_count, boosts_count, likes_count, type, interaction, poll_json, created_at)
          VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?::jsonb,?,?,?,?,?,?,?,?,?::jsonb,?)
          ON CONFLICT (id) DO UPDATE SET kind=EXCLUDED.kind, content=EXCLUDED.content, visibility=EXCLUDED.visibility,
            instance=EXCLUDED.instance, storage_uri=EXCLUDED.storage_uri, attestation_uri=EXCLUDED.attestation_uri,
            chain_id=EXCLUDED.chain_id, tx_hash=EXCLUDED.tx_hash, contract_address=EXCLUDED.contract_address, explorer_url=EXCLUDED.explorer_url,
            tags_json=EXCLUDED.tags_json, parent_post_id=EXCLUDED.parent_post_id, root_post_id=EXCLUDED.root_post_id,
            reply_depth=EXCLUDED.reply_depth, replies_count=EXCLUDED.replies_count, boosts_count=EXCLUDED.boosts_count,
            likes_count=EXCLUDED.likes_count, type=EXCLUDED.type, interaction=EXCLUDED.interaction, poll_json=EXCLUDED.poll_json
          """)) {
        ps.setString(1, post.id);
        ps.setString(2, post.authorId);
        ps.setString(3, Strings.value(post.instance));
        ps.setString(4, post.kind);
        ps.setString(5, Strings.value(post.content));
        ps.setString(6, Strings.or(post.visibility, "public"));
        ps.setString(7, Strings.value(post.storageUri));
        ps.setString(8, Strings.value(post.attestationUri));
        ps.setString(9, Strings.value(post.chainId));
        ps.setString(10, Strings.value(post.txHash));
        ps.setString(11, Strings.value(post.contractAddress));
        ps.setString(12, Strings.value(post.explorerUrl));
        ps.setString(13, json(post.tags));
        setNullable(ps, 14, post.parentPostId);
        setNullable(ps, 15, post.rootPostId);
        ps.setInt(16, post.replyDepth);
        ps.setInt(17, post.replies);
        ps.setInt(18, post.boosts);
        ps.setInt(19, post.likes);
        ps.setString(20, Strings.or(post.type, "post"));
        ps.setString(21, Strings.or(post.interaction, "anyone"));
        ps.setString(22, post.poll == null ? "null" : json(post.poll));
        ps.setTimestamp(23, timestamp(post.createdAt));
        ps.executeUpdate();
      }
      try (PreparedStatement delete = conn.prepareStatement("DELETE FROM post_media_links WHERE post_id = ?")) {
        delete.setString(1, post.id);
        delete.executeUpdate();
      }
      if (post.media != null) {
        for (PostMedia item : post.media) {
          try (PreparedStatement link = conn.prepareStatement("INSERT INTO post_media_links(post_id, media_id) VALUES (?,?) ON CONFLICT DO NOTHING")) {
            link.setString(1, post.id);
            link.setString(2, item.id);
            link.executeUpdate();
          }
        }
      }
    });
  }

  void saveFollow(String followerID, String targetID) {
    if (!databaseAvailable) return;
    exec(conn -> {
      try (PreparedStatement ps = conn.prepareStatement("INSERT INTO user_follows(follower_id, followee_id) VALUES (?,?) ON CONFLICT DO NOTHING")) {
        ps.setString(1, followerID);
        ps.setString(2, targetID);
        ps.executeUpdate();
      }
    });
  }

  void deleteFollow(String followerID, String targetID) {
    if (!databaseAvailable) return;
    exec(conn -> {
      try (PreparedStatement ps = conn.prepareStatement("DELETE FROM user_follows WHERE follower_id=? AND followee_id=?")) {
        ps.setString(1, followerID);
        ps.setString(2, targetID);
        ps.executeUpdate();
      }
    });
  }

  void saveConversation(Conversation conversation) {
    if (!databaseAvailable) return;
    exec(conn -> {
      try (PreparedStatement ps = conn.prepareStatement("""
          INSERT INTO conversations(id, title, initiator_id, encrypted, asset_uri, chain_id,
            tx_hash, contract_address, explorer_url, updated_at)
          VALUES (?,?,?,?,?,?,?,?,?,?)
          ON CONFLICT (id) DO UPDATE SET title=EXCLUDED.title, initiator_id=EXCLUDED.initiator_id,
            encrypted=EXCLUDED.encrypted, asset_uri=EXCLUDED.asset_uri, chain_id=EXCLUDED.chain_id,
            tx_hash=EXCLUDED.tx_hash, contract_address=EXCLUDED.contract_address,
            explorer_url=EXCLUDED.explorer_url, updated_at=EXCLUDED.updated_at
          """)) {
        ps.setString(1, conversation.id);
        ps.setString(2, conversation.title);
        setNullable(ps, 3, conversation.initiatorId);
        ps.setBoolean(4, conversation.encrypted);
        ps.setString(5, Strings.value(conversation.assetUri));
        ps.setString(6, Strings.value(conversation.chainId));
        ps.setString(7, Strings.value(conversation.txHash));
        ps.setString(8, Strings.value(conversation.contractAddress));
        ps.setString(9, Strings.value(conversation.explorerUrl));
        ps.setTimestamp(10, timestamp(conversation.updatedAt));
        ps.executeUpdate();
      }
      try (PreparedStatement deleteParticipants = conn.prepareStatement("DELETE FROM conversation_participants WHERE conversation_id=?")) {
        deleteParticipants.setString(1, conversation.id);
        deleteParticipants.executeUpdate();
      }
      for (String participantID : conversation.participantIds) {
        try (PreparedStatement ps = conn.prepareStatement("INSERT INTO conversation_participants(conversation_id, user_id) VALUES (?,?) ON CONFLICT DO NOTHING")) {
          ps.setString(1, conversation.id);
          ps.setString(2, participantID);
          ps.executeUpdate();
        }
      }
      try (PreparedStatement deleteMessages = conn.prepareStatement("DELETE FROM chat_messages WHERE conversation_id=?")) {
        deleteMessages.setString(1, conversation.id);
        deleteMessages.executeUpdate();
      }
      for (ChatMessage message : conversation.messages) {
        try (PreparedStatement ps = conn.prepareStatement("""
            INSERT INTO chat_messages(id, conversation_id, sender_id, body, asset_uri, chain_id,
              tx_hash, contract_address, explorer_url, created_at)
            VALUES (?,?,?,?,?,?,?,?,?,?)
            ON CONFLICT (id) DO UPDATE SET body=EXCLUDED.body, asset_uri=EXCLUDED.asset_uri,
              chain_id=EXCLUDED.chain_id, tx_hash=EXCLUDED.tx_hash,
              contract_address=EXCLUDED.contract_address, explorer_url=EXCLUDED.explorer_url
            """)) {
          ps.setString(1, message.id);
          ps.setString(2, message.conversationId);
          ps.setString(3, message.senderId);
          ps.setString(4, message.body);
          ps.setString(5, Strings.value(message.assetUri));
          ps.setString(6, Strings.value(message.chainId));
          ps.setString(7, Strings.value(message.txHash));
          ps.setString(8, Strings.value(message.contractAddress));
          ps.setString(9, Strings.value(message.explorerUrl));
          ps.setTimestamp(10, timestamp(message.createdAt));
          ps.executeUpdate();
        }
      }
    });
  }

  private void initDatabase() {
    if (!Strings.hasText(databaseUrl)) return;
    try (Connection conn = connection()) {
      databaseAvailable = true;
      applyMigrations(conn);
    } catch (Exception err) {
      databaseAvailable = false;
    }
  }

  private void initRedis() {
    String addr = Strings.or(Env.get("REDIS_ADDR", ""), "127.0.0.1:6379");
    try {
      String[] parts = addr.split(":", 2);
      redis = new JedisPooled(parts[0], parts.length > 1 ? Integer.parseInt(parts[1]) : 6379);
      redis.ping();
    } catch (Exception err) {
      redis = null;
    }
  }

  private void applyMigrations(Connection conn) throws SQLException, IOException {
    try (Statement st = conn.createStatement()) {
      st.executeUpdate("CREATE TABLE IF NOT EXISTS schema_migrations(version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())");
    }
    Set<String> applied = new HashSet<>();
    try (PreparedStatement ps = conn.prepareStatement("SELECT version FROM schema_migrations ORDER BY version ASC")) {
      ResultSet rs = ps.executeQuery();
      while (rs.next()) {
        applied.add(rs.getString(1));
        appliedMigrations.add(rs.getString(1));
      }
    }
    Path dir = Path.of(migrationsDir);
    if (!dir.isAbsolute()) dir = Path.of(".").resolve(dir).normalize();
    if (!Files.isDirectory(dir)) return;
    List<Path> files;
    try (var stream = Files.list(dir)) {
      files = stream.filter(path -> path.getFileName().toString().endsWith(".sql")).sorted().toList();
    }
    for (Path file : files) {
      String version = file.getFileName().toString();
      migrationFiles.add(version);
      if (applied.contains(version)) continue;
      try (Statement st = conn.createStatement()) {
        st.execute(Files.readString(file));
      }
      try (PreparedStatement ps = conn.prepareStatement("INSERT INTO schema_migrations(version) VALUES (?) ON CONFLICT DO NOTHING")) {
        ps.setString(1, version);
        ps.executeUpdate();
      }
      appliedMigrations.add(version);
    }
    ensureSchemaCompatibility(conn);
  }

  private void ensureSchemaCompatibility(Connection conn) throws SQLException {
    try (Statement st = conn.createStatement()) {
      st.executeUpdate("ALTER TABLE social_posts ADD COLUMN IF NOT EXISTS chain_id TEXT NOT NULL DEFAULT ''");
      st.executeUpdate("ALTER TABLE social_posts ADD COLUMN IF NOT EXISTS instance TEXT NOT NULL DEFAULT ''");
      st.executeUpdate("ALTER TABLE social_posts ADD COLUMN IF NOT EXISTS tx_hash TEXT NOT NULL DEFAULT ''");
      st.executeUpdate("ALTER TABLE social_posts ADD COLUMN IF NOT EXISTS contract_address TEXT NOT NULL DEFAULT ''");
      st.executeUpdate("ALTER TABLE social_posts ADD COLUMN IF NOT EXISTS explorer_url TEXT NOT NULL DEFAULT ''");
      st.executeUpdate("ALTER TABLE social_posts ADD COLUMN IF NOT EXISTS poll_json JSONB");
      st.executeUpdate("ALTER TABLE social_users ADD COLUMN IF NOT EXISTS fields_json JSONB NOT NULL DEFAULT '[]'::jsonb");
      st.executeUpdate("ALTER TABLE social_users ADD COLUMN IF NOT EXISTS featured_tags_json JSONB NOT NULL DEFAULT '[]'::jsonb");
      st.executeUpdate("ALTER TABLE social_users ADD COLUMN IF NOT EXISTS is_bot BOOLEAN NOT NULL DEFAULT FALSE");
      st.executeUpdate("ALTER TABLE auth_accounts ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active'");
      st.executeUpdate("ALTER TABLE conversations ADD COLUMN IF NOT EXISTS initiator_id TEXT");
      st.executeUpdate("ALTER TABLE conversations ADD COLUMN IF NOT EXISTS encrypted BOOLEAN NOT NULL DEFAULT FALSE");
      st.executeUpdate("ALTER TABLE conversations ADD COLUMN IF NOT EXISTS asset_uri TEXT NOT NULL DEFAULT ''");
      st.executeUpdate("ALTER TABLE conversations ADD COLUMN IF NOT EXISTS chain_id TEXT NOT NULL DEFAULT ''");
      st.executeUpdate("ALTER TABLE conversations ADD COLUMN IF NOT EXISTS tx_hash TEXT NOT NULL DEFAULT ''");
      st.executeUpdate("ALTER TABLE conversations ADD COLUMN IF NOT EXISTS contract_address TEXT NOT NULL DEFAULT ''");
      st.executeUpdate("ALTER TABLE conversations ADD COLUMN IF NOT EXISTS explorer_url TEXT NOT NULL DEFAULT ''");
      st.executeUpdate("ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS asset_uri TEXT NOT NULL DEFAULT ''");
      st.executeUpdate("ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS chain_id TEXT NOT NULL DEFAULT ''");
      st.executeUpdate("ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS tx_hash TEXT NOT NULL DEFAULT ''");
      st.executeUpdate("ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS contract_address TEXT NOT NULL DEFAULT ''");
      st.executeUpdate("ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS explorer_url TEXT NOT NULL DEFAULT ''");
    }
  }

  private SocialState loadSocialFromPostgres() {
    SocialState state = new SocialState();
    query(conn -> {
      try (PreparedStatement ps = conn.prepareStatement("SELECT name, focus, members, latency, status FROM federation_instances ORDER BY name")) {
        ResultSet rs = ps.executeQuery();
        while (rs.next()) state.instances.add(new FederationInstance(rs.getString(1), rs.getString(2), rs.getString(3), rs.getString(4), rs.getString(5)));
      }
      try (PreparedStatement ps = conn.prepareStatement("SELECT id, handle, display_name, bio, instance, wallet, avatar_url, fields_json, featured_tags_json, is_bot, followers_count, following_count, created_at FROM social_users ORDER BY created_at DESC")) {
        ResultSet rs = ps.executeQuery();
        while (rs.next()) {
          SocialUser user = new SocialUser();
          user.id = rs.getString("id");
          user.handle = rs.getString("handle");
          user.displayName = rs.getString("display_name");
          user.bio = rs.getString("bio");
          user.instance = rs.getString("instance");
          user.wallet = rs.getString("wallet");
          user.avatarUrl = rs.getString("avatar_url");
          user.fields = parse(rs.getString("fields_json"), new TypeReference<List<UserField>>() {});
          user.featuredTags = parse(rs.getString("featured_tags_json"), new TypeReference<List<String>>() {});
          user.isBot = rs.getBoolean("is_bot");
          user.followers = rs.getInt("followers_count");
          user.following = rs.getInt("following_count");
          user.createdAt = instantString(rs.getTimestamp("created_at"));
          state.users.add(user);
        }
      }
      Map<String, MediaAsset> mediaByID = new HashMap<>();
      try (PreparedStatement ps = conn.prepareStatement("SELECT id, owner_id, name, kind, url, storage_uri, cid, size_bytes, status, created_at FROM media_assets ORDER BY created_at DESC")) {
        ResultSet rs = ps.executeQuery();
        while (rs.next()) {
          MediaAsset asset = new MediaAsset(rs.getString("id"), rs.getString("owner_id"), rs.getString("name"), rs.getString("kind"), rs.getString("url"), rs.getString("storage_uri"), rs.getString("cid"), rs.getLong("size_bytes"), rs.getString("status"), instantString(rs.getTimestamp("created_at")));
          mediaByID.put(asset.id, asset);
          state.media.add(asset);
        }
      }
      try (PreparedStatement ps = conn.prepareStatement("""
          SELECT p.*, u.handle, u.display_name, COALESCE(NULLIF(p.instance, ''), u.instance) AS post_instance
          FROM social_posts p JOIN social_users u ON u.id = p.author_id
          ORDER BY p.created_at DESC
          """)) {
        ResultSet rs = ps.executeQuery();
        while (rs.next()) {
          SocialPost post = new SocialPost();
          post.id = rs.getString("id");
          post.authorId = rs.getString("author_id");
          post.authorHandle = rs.getString("handle");
          post.authorName = rs.getString("display_name");
          post.instance = rs.getString("post_instance");
          post.kind = rs.getString("kind");
          post.content = rs.getString("content");
          post.visibility = rs.getString("visibility");
          post.storageUri = rs.getString("storage_uri");
          post.attestationUri = rs.getString("attestation_uri");
          post.chainId = rs.getString("chain_id");
          post.txHash = rs.getString("tx_hash");
          post.contractAddress = rs.getString("contract_address");
          post.explorerUrl = rs.getString("explorer_url");
          post.tags = parse(rs.getString("tags_json"), new TypeReference<List<String>>() {});
          post.parentPostId = Strings.value(rs.getString("parent_post_id"));
          post.rootPostId = Strings.value(rs.getString("root_post_id"));
          post.replyDepth = rs.getInt("reply_depth");
          post.replies = rs.getInt("replies_count");
          post.boosts = rs.getInt("boosts_count");
          post.likes = rs.getInt("likes_count");
          post.type = rs.getString("type");
          post.interaction = rs.getString("interaction");
          String pollJson = rs.getString("poll_json");
          if (Strings.hasText(pollJson) && !"null".equals(pollJson)) post.poll = parse(pollJson, new TypeReference<Poll>() {});
          post.createdAt = instantString(rs.getTimestamp("created_at"));
          post.media = loadPostMedia(conn, post.id, mediaByID);
          state.posts.add(post);
        }
      }
      try (PreparedStatement ps = conn.prepareStatement("SELECT follower_id, followee_id FROM user_follows")) {
        ResultSet rs = ps.executeQuery();
        while (rs.next()) state.follows.computeIfAbsent(rs.getString(1), key -> new HashSet<>()).add(rs.getString(2));
      }
      try (PreparedStatement ps = conn.prepareStatement("SELECT id, title, initiator_id, encrypted, asset_uri, chain_id, tx_hash, contract_address, explorer_url, updated_at FROM conversations ORDER BY updated_at DESC")) {
        ResultSet rs = ps.executeQuery();
        while (rs.next()) {
          Conversation item = new Conversation();
          item.id = rs.getString("id");
          item.title = rs.getString("title");
          item.initiatorId = Strings.value(rs.getString("initiator_id"));
          item.encrypted = rs.getBoolean("encrypted");
          item.assetUri = rs.getString("asset_uri");
          item.chainId = rs.getString("chain_id");
          item.txHash = rs.getString("tx_hash");
          item.contractAddress = rs.getString("contract_address");
          item.explorerUrl = rs.getString("explorer_url");
          item.updatedAt = instantString(rs.getTimestamp("updated_at"));
          item.participantIds = loadParticipants(conn, item.id);
          item.messages = loadMessages(conn, item.id);
          state.conversations.add(item);
        }
      }
    });
    return state;
  }

  private List<PostMedia> loadPostMedia(Connection conn, String postID, Map<String, MediaAsset> mediaByID) throws SQLException {
    List<PostMedia> result = new ArrayList<>();
    try (PreparedStatement ps = conn.prepareStatement("SELECT media_id FROM post_media_links WHERE post_id=?")) {
      ps.setString(1, postID);
      ResultSet rs = ps.executeQuery();
      while (rs.next()) {
        MediaAsset asset = mediaByID.get(rs.getString(1));
        if (asset != null) result.add(PostMedia.from(asset));
      }
    }
    return result;
  }

  private List<String> loadParticipants(Connection conn, String conversationID) throws SQLException {
    List<String> result = new ArrayList<>();
    try (PreparedStatement ps = conn.prepareStatement("SELECT user_id FROM conversation_participants WHERE conversation_id=? ORDER BY user_id ASC")) {
      ps.setString(1, conversationID);
      ResultSet rs = ps.executeQuery();
      while (rs.next()) result.add(rs.getString(1));
    }
    return result;
  }

  private List<ChatMessage> loadMessages(Connection conn, String conversationID) throws SQLException {
    List<ChatMessage> result = new ArrayList<>();
    try (PreparedStatement ps = conn.prepareStatement("""
        SELECT m.id, m.conversation_id, m.sender_id, u.handle, m.body, m.asset_uri, m.chain_id,
          m.tx_hash, m.contract_address, m.explorer_url, m.created_at
        FROM chat_messages m JOIN social_users u ON u.id = m.sender_id
        WHERE m.conversation_id=? ORDER BY m.created_at ASC
        """)) {
      ps.setString(1, conversationID);
      ResultSet rs = ps.executeQuery();
      while (rs.next()) {
        ChatMessage message = new ChatMessage(rs.getString(1), rs.getString(2), rs.getString(3), rs.getString(4), rs.getString(5), instantString(rs.getTimestamp(11)));
        message.assetUri = rs.getString(6);
        message.chainId = rs.getString(7);
        message.txHash = rs.getString(8);
        message.contractAddress = rs.getString(9);
        message.explorerUrl = rs.getString(10);
        result.add(message);
      }
    }
    return result;
  }

  private Connection connection() throws SQLException {
    return DriverManager.getConnection(toJdbcUrl(databaseUrl));
  }

  private void exec(SqlConsumer consumer) {
    if (!databaseAvailable) return;
    try (Connection conn = connection()) {
      consumer.accept(conn);
    } catch (Exception err) {
      throw new RuntimeException(err);
    }
  }

  private void query(SqlConsumer consumer) {
    exec(consumer);
  }

  private <T> T readJson(String key, TypeReference<T> type) {
    try {
      String raw = redis.get(key);
      if (!Strings.hasText(raw)) throw new IllegalStateException("missing redis key: " + key);
      return JSON.readValue(raw, type);
    } catch (Exception err) {
      throw new RuntimeException(err);
    }
  }

  private void setJson(String key, Object value, int ttlSeconds) {
    try {
      String raw = JSON.writeValueAsString(value);
      if (ttlSeconds > 0) redis.setex(key, ttlSeconds, raw);
      else redis.set(key, raw);
    } catch (Exception ignored) {
    }
  }

  private static String json(Object value) {
    try {
      return JSON.writeValueAsString(value);
    } catch (Exception err) {
      return "null";
    }
  }

  private static <T> T parse(String raw, TypeReference<T> type) {
    try {
      return JSON.readValue(Strings.hasText(raw) ? raw : "null", type);
    } catch (Exception err) {
      try {
        return JSON.readValue("[]", type);
      } catch (Exception ignored) {
        return null;
      }
    }
  }

  private static Timestamp timestamp(String raw) {
    try {
      return Timestamp.from(Instant.parse(raw));
    } catch (Exception err) {
      return Timestamp.from(Instant.now());
    }
  }

  private static String instantString(Timestamp timestamp) {
    return timestamp == null ? "" : timestamp.toInstant().toString();
  }

  private static void setNullable(PreparedStatement ps, int index, String value) throws SQLException {
    if (Strings.hasText(value)) ps.setString(index, value);
    else ps.setNull(index, Types.VARCHAR);
  }

  private static String toJdbcUrl(String raw) {
    if (raw.startsWith("jdbc:")) return raw;
    if (raw.startsWith("postgres://")) {
      try {
        URI uri = URI.create(raw);
        String userInfo = uri.getUserInfo();
        String query = Strings.value(uri.getRawQuery());
        if (Strings.hasText(userInfo)) {
          String[] parts = userInfo.split(":", 2);
          query = appendQuery(query, "user=" + urlDecode(parts[0]));
          if (parts.length > 1) query = appendQuery(query, "password=" + urlDecode(parts[1]));
        }
        return "jdbc:postgresql://" + uri.getHost() + (uri.getPort() > 0 ? ":" + uri.getPort() : "")
            + uri.getPath() + (Strings.hasText(query) ? "?" + query : "");
      } catch (Exception ignored) {
        return raw.replaceFirst("^postgres://", "jdbc:postgresql://");
      }
    }
    if (raw.startsWith("postgresql://")) {
      return raw.replaceFirst("^postgresql://", "jdbc:postgresql://");
    }
    return raw;
  }

  private static String appendQuery(String query, String item) {
    return Strings.hasText(query) ? query + "&" + item : item;
  }

  private static String urlDecode(String value) {
    return URLDecoder.decode(value, StandardCharsets.UTF_8);
  }

  @FunctionalInterface
  interface SqlConsumer {
    void accept(Connection conn) throws Exception;
  }
}

class SocialState {
  public List<SocialUser> users = new ArrayList<>();
  public List<SocialPost> posts = new ArrayList<>();
  public List<MediaAsset> media = new ArrayList<>();
  public List<Conversation> conversations = new ArrayList<>();
  public List<FederationInstance> instances = new ArrayList<>();
  public Map<String, Set<String>> follows = new HashMap<>();
}

@Service
class ActiveSessionRegistry {
  private final Map<String, AuthSessionRecord> sessions = new ConcurrentHashMap<>();

  void touch(AuthSessionRecord session) {
    if (session != null && Strings.hasText(session.id) && !expired(session.expiresAt)) {
      sessions.put(session.id, session);
    }
  }

  void remove(String sessionID) {
    if (Strings.hasText(sessionID)) {
      sessions.remove(sessionID);
    }
  }

  Map<String, Integer> countOnlineByInstance(List<SocialUser> users) {
    Instant now = Instant.now();
    Map<String, SocialUser> usersByID = new HashMap<>();
    for (SocialUser user : users) {
      usersByID.put(user.id, user);
    }

    Set<String> countedUsers = new HashSet<>();
    Map<String, Integer> result = new HashMap<>();
    for (AuthSessionRecord session : sessions.values()) {
      if (!Strings.hasText(session.expiresAt)) continue;
      Instant expiresAt;
      try {
        expiresAt = Instant.parse(session.expiresAt);
      } catch (DateTimeParseException err) {
        continue;
      }
      if (!expiresAt.isAfter(now)) {
        sessions.remove(session.id);
        continue;
      }
      if (!countedUsers.add(session.userId)) continue;
      SocialUser user = usersByID.get(session.userId);
      if (user != null) {
        result.merge(user.instance, 1, Integer::sum);
      }
    }
    return result;
  }

  private static boolean expired(String timestamp) {
    try {
      return Instant.now().isAfter(Instant.parse(timestamp));
    } catch (DateTimeParseException err) {
      return true;
    }
  }
}

@Service
class AuthService {
  static final String SESSION_COOKIE = "molesociety_session";
  private static final Duration CHALLENGE_TTL = Duration.ofMinutes(5);
  private static final Duration SESSION_TTL = Duration.ofDays(7);

  private final SocialService social;
  private final PersistenceService persistence;
  private final ActiveSessionRegistry activeSessions;
  private final BCryptPasswordEncoder encoder = new BCryptPasswordEncoder();
  private final SecureRandom random = new SecureRandom();
  private final Map<String, AuthChallenge> challenges = new ConcurrentHashMap<>();
  private final Map<String, AuthSessionRecord> sessions = new ConcurrentHashMap<>();
  private final Map<String, Account> accounts = new ConcurrentHashMap<>();

  AuthService(SocialService social, PersistenceService persistence, ActiveSessionRegistry activeSessions) {
    this.social = social;
    this.persistence = persistence;
    this.activeSessions = activeSessions;
    accounts.putAll(persistence.loadAuthAccounts());
    ensureAccountsForUsers();
  }

  AuthChallenge createChallenge(String address, long chainID, String uri) {
    String wallet = normalizeWallet(address);
    long resolvedChainID = chainID == 0 ? Env.longValue("CHAIN_ID", 1L) : chainID;
    Instant issued = Instant.now();
    Instant expires = issued.plus(CHALLENGE_TTL);
    String nonce = randomHex(16);
    AuthChallenge challenge = new AuthChallenge();
    challenge.nonce = nonce;
    challenge.address = wallet;
    challenge.chainId = resolvedChainID;
    challenge.issuedAt = issued.toString();
    challenge.expiresAt = expires.toString();
    challenge.message = "MoleSociety wants you to sign in with your wallet:\n" + wallet
        + "\n\nSign in to MoleSociety.\n\nURI: " + uri
        + "\nVersion: 1\nChain ID: " + resolvedChainID
        + "\nNonce: " + nonce
        + "\nIssued At: " + challenge.issuedAt
        + "\nExpiration Time: " + challenge.expiresAt;
    challenges.put(nonce, challenge);
    persistence.saveChallenge(challenge, CHALLENGE_TTL);
    return challenge;
  }

  AuthResult verifyWalletLogin(String address, String nonce, String signature) {
    AuthChallenge challenge = challenge(nonce);
    String wallet = normalizeWallet(address);
    if (!wallet.equalsIgnoreCase(challenge.address)) {
      throw ApiException.unauthorized("AUTH_WALLET_CHALLENGE_MISMATCH", "wallet", "wallet address does not match login challenge");
    }
    if (!Strings.hasText(signature)) {
      throw ApiException.unauthorized("AUTH_INVALID_SIGNATURE", "wallet", "wallet signature is required");
    }
    Account account = accounts.values().stream()
        .filter(a -> wallet.equalsIgnoreCase(a.wallet))
        .findFirst()
        .orElseThrow(() -> ApiException.notFound("AUTH_WALLET_NOT_BOUND", "wallet", "wallet is not bound to any account, please register first"));
    SocialUser user = social.getUser(account.userId);
    AuthSessionRecord session = createSession(user.id, account.wallet);
    challenges.remove(nonce);
    persistence.deleteChallenge(nonce);
    return new AuthResult(user, session);
  }

  AuthResult passwordLogin(String identifier, String password) {
    if (!Strings.hasText(identifier) || !Strings.hasText(password)) {
      throw ApiException.badRequest("AUTH_MISSING_CREDENTIALS", "validation", "identifier and password are required");
    }
    String key = identifier.trim().toLowerCase(Locale.ROOT);
    Account account = accounts.values().stream()
        .filter(a -> key.equals(a.username.toLowerCase(Locale.ROOT)) || key.equals(Strings.value(a.email).toLowerCase(Locale.ROOT)))
        .findFirst()
        .orElseThrow(() -> ApiException.notFound("AUTH_ACCOUNT_NOT_FOUND", "credentials", "account not found"));
    if (!encoder.matches(password, account.passwordHash)) {
      throw ApiException.unauthorized("AUTH_INVALID_PASSWORD", "credentials", "invalid password");
    }
    SocialUser user = social.getUser(account.userId);
    return new AuthResult(user, createSession(user.id, account.wallet));
  }

  synchronized AuthResult register(RegisterRequest req, String uri) {
    String username = Strings.value(req.username).trim();
    String password = Strings.value(req.password).trim();
    if (!Strings.hasText(username) || !Strings.hasText(password)) {
      throw ApiException.badRequest("AUTH_MISSING_REGISTRATION_FIELDS", "validation", "username and password are required");
    }
    if (password.length() < 6) {
      throw ApiException.badRequest("AUTH_WEAK_PASSWORD", "validation", "password must be at least 6 characters");
    }
    for (Account existing : accounts.values()) {
      if (username.equalsIgnoreCase(existing.username)) {
        throw ApiException.conflict("AUTH_USERNAME_TAKEN", "conflict", "username already taken");
      }
      if (Strings.hasText(req.email) && req.email.equalsIgnoreCase(existing.email)) {
        throw ApiException.conflict("AUTH_EMAIL_TAKEN", "conflict", "email already registered");
      }
    }

    String wallet = req.autoWallet || !Strings.hasText(req.walletAddress) ? createAutoWallet() : normalizeWallet(req.walletAddress);
    for (Account existing : accounts.values()) {
      if (wallet.equalsIgnoreCase(existing.wallet)) {
        throw ApiException.conflict("AUTH_WALLET_ALREADY_BOUND", "conflict", "wallet already bound to another account");
      }
    }
    if (!req.autoWallet && Strings.hasText(req.walletAddress)) {
      AuthChallenge challenge = challenge(req.nonce);
      if (!wallet.equalsIgnoreCase(challenge.address) || !Strings.hasText(req.signature)) {
        throw ApiException.badRequest("AUTH_BIND_SIGNATURE_REQUIRED", "validation", "wallet binding nonce and signature are required");
      }
      challenges.remove(req.nonce);
      persistence.deleteChallenge(req.nonce);
    }

    SocialUser user = social.createUser(new CreateUserRequest(username, username, "MoleSociety member", "摩尔1号", wallet, ""));
    Account account = new Account();
    account.id = "acc_" + System.nanoTime();
    account.username = username;
    account.email = Strings.hasText(req.email) ? req.email.trim() : username + "@local.molesociety";
    account.passwordHash = encoder.encode(password);
    account.wallet = wallet;
    account.userId = user.id;
    account.status = "active";
    account.createdAt = Instant.now().toString();
    account.updatedAt = account.createdAt;
    accounts.put(account.id, account);
    persistence.saveAuthAccount(account);
    return new AuthResult(user, createSession(user.id, wallet));
  }

  Optional<SocialUser> userFromRequest(HttpServletRequest request) {
    Optional<String> cookie = AuthController.sessionCookie(request);
    if (cookie.isEmpty()) {
      return Optional.empty();
    }
    AuthSessionRecord session = persistence.loadSession(cookie.get()).orElse(sessions.get(cookie.get()));
    if (session == null || expired(session.expiresAt)) {
      sessions.remove(cookie.get());
      activeSessions.remove(cookie.get());
      throw ApiException.unauthorized("AUTH_SESSION_INVALID", "session", "invalid session");
    }
    activeSessions.touch(session);
    return Optional.of(social.getUser(session.userId));
  }

  void deleteSession(String sessionID) {
    sessions.remove(sessionID);
    activeSessions.remove(sessionID);
    persistence.deleteSession(sessionID);
  }

  private void ensureAccountsForUsers() {
    Set<String> accountUserIDs = new HashSet<>();
    Set<String> usernames = new HashSet<>();
    Set<String> emails = new HashSet<>();
    Set<String> wallets = new HashSet<>();
    for (Account account : accounts.values()) {
      accountUserIDs.add(account.userId);
      usernames.add(Strings.value(account.username).toLowerCase(Locale.ROOT));
      emails.add(Strings.value(account.email).toLowerCase(Locale.ROOT));
      wallets.add(Strings.value(account.wallet).toLowerCase(Locale.ROOT));
    }

    for (SocialUser user : social.listUsers()) {
      if (accountUserIDs.contains(user.id)) continue;

      Account account = new Account();
      account.id = "acc_backfill_" + user.id.replaceAll("[^A-Za-z0-9_\\-]", "_");
      account.username = uniqueValue(baseUsername(user), usernames);
      account.email = uniqueValue(account.username + "@local.molesociety", emails);
      account.passwordHash = encoder.encode(System.getProperty("BACKFILL_ACCOUNT_PASSWORD", "123456"));
      account.wallet = uniqueValue(Strings.value(user.wallet), wallets);
      account.userId = user.id;
      account.status = "active";
      account.createdAt = Instant.now().toString();
      account.updatedAt = account.createdAt;
      accounts.put(account.id, account);
      accountUserIDs.add(account.userId);
      persistence.saveAuthAccount(account);
    }
  }

  private static String baseUsername(SocialUser user) {
    String base = Strings.value(user.handle).replaceFirst("^@", "").trim();
    base = base.replaceAll("[^A-Za-z0-9_\\-]", "_");
    return Strings.hasText(base) ? base : user.id;
  }

  private static String uniqueValue(String base, Set<String> used) {
    String value = Strings.hasText(base) ? base.trim() : "user";
    String candidate = value;
    int suffix = 2;
    while (used.contains(candidate.toLowerCase(Locale.ROOT))) {
      candidate = value + "_" + suffix++;
    }
    used.add(candidate.toLowerCase(Locale.ROOT));
    return candidate;
  }

  private AuthChallenge challenge(String nonce) {
    if (!Strings.hasText(nonce)) {
      throw ApiException.badRequest("AUTH_NONCE_REQUIRED", "validation", "nonce is required");
    }
    AuthChallenge challenge = persistence.loadChallenge(nonce.trim()).orElse(challenges.get(nonce.trim()));
    if (challenge == null || expired(challenge.expiresAt)) {
      challenges.remove(nonce.trim());
      throw ApiException.unauthorized("AUTH_CHALLENGE_EXPIRED", "wallet", "login challenge expired or missing");
    }
    return challenge;
  }

  private AuthSessionRecord createSession(String userID, String wallet) {
    AuthSessionRecord session = new AuthSessionRecord();
    session.id = randomHex(32);
    session.userId = userID;
    session.address = wallet;
    session.createdAt = Instant.now().toString();
    session.expiresAt = Instant.now().plus(SESSION_TTL).toString();
    sessions.put(session.id, session);
    activeSessions.touch(session);
    persistence.saveSession(session, SESSION_TTL);
    return session;
  }

  private String createAutoWallet() {
    byte[] bytes = new byte[20];
    random.nextBytes(bytes);
    return "0x" + bytesToHex(bytes);
  }

  private String randomHex(int bytes) {
    byte[] token = new byte[bytes];
    random.nextBytes(token);
    return bytesToHex(token);
  }

  private static String normalizeWallet(String address) {
    String trimmed = Strings.value(address).trim();
    if (!trimmed.matches("(?i)^0x[0-9a-f]{40}$")) {
      throw ApiException.badRequest("AUTH_INVALID_WALLET", "wallet", "invalid wallet address");
    }
    return checksumAddress(trimmed);
  }

  private static String checksumAddress(String address) {
    String lower = address.replace("0x", "").replace("0X", "").toLowerCase(Locale.ROOT);
    Keccak.Digest256 digest = new Keccak.Digest256();
    byte[] hash = digest.digest(lower.getBytes());
    BigInteger hashInt = new BigInteger(1, hash);
    String hashHex = String.format("%064x", hashInt);
    StringBuilder out = new StringBuilder("0x");
    for (int i = 0; i < lower.length(); i++) {
      char c = lower.charAt(i);
      if (Character.digit(hashHex.charAt(i), 16) >= 8) {
        out.append(Character.toUpperCase(c));
      } else {
        out.append(c);
      }
    }
    return out.toString();
  }

  private static String bytesToHex(byte[] bytes) {
    StringBuilder out = new StringBuilder(bytes.length * 2);
    for (byte b : bytes) {
      out.append(String.format("%02x", b));
    }
    return out.toString();
  }

  private static boolean expired(String timestamp) {
    try {
      return Instant.now().isAfter(Instant.parse(timestamp));
    } catch (DateTimeParseException err) {
      return true;
    }
  }
}

@Service
class SocialService {
  private final PersistenceService persistence;
  private final ActiveSessionRegistry activeSessions;
  private final List<SocialUser> users = new ArrayList<>();
  private final List<SocialPost> posts = new ArrayList<>();
  private final List<MediaAsset> media = new ArrayList<>();
  private final List<Conversation> conversations = new ArrayList<>();
  private final List<FederationInstance> instances = new ArrayList<>();
  private final Map<String, Set<String>> follows = new HashMap<>();

  SocialService(PersistenceService persistence, ActiveSessionRegistry activeSessions) {
    this.persistence = persistence;
    this.activeSessions = activeSessions;
    if (!persistence.loadSocialState(this)) {
      reset(Env.truthy(Env.get("SOCIAL_SEED", persistence.databaseAvailable() ? "0" : "1")));
    } else {
      normalizeMoleInstances();
      ensureChainAssets();
      persistence.saveLoadedSocialState(snapshot());
    }
  }

  synchronized void reset(boolean seed) {
    users.clear();
    posts.clear();
    media.clear();
    conversations.clear();
    instances.clear();
    follows.clear();
    if (seed) {
      seed();
    }
    ensureChainAssets();
    persistence.replaceSocialState(snapshot());
  }

  synchronized void replaceState(SocialState state) {
    users.clear();
    posts.clear();
    media.clear();
    conversations.clear();
    instances.clear();
    follows.clear();
    users.addAll(state.users);
    posts.addAll(state.posts);
    media.addAll(state.media);
    conversations.addAll(state.conversations);
    instances.addAll(state.instances);
    follows.putAll(state.follows);
    ensureChainAssets();
    refreshPostCounts();
    refreshFollowCounts();
  }

  synchronized SocialState snapshot() {
    SocialState state = new SocialState();
    state.users.addAll(users);
    state.posts.addAll(posts);
    state.media.addAll(media);
    state.conversations.addAll(conversations);
    state.instances.addAll(instances);
    for (Map.Entry<String, Set<String>> entry : follows.entrySet()) {
      state.follows.put(entry.getKey(), new HashSet<>(entry.getValue()));
    }
    return state;
  }

  synchronized SocialStats stats() {
    return new SocialStats(users.size(), posts.size(), media.size(), conversations.size());
  }

  synchronized BootstrapPayload bootstrap(int limit, String currentUserID, boolean mine) {
    BootstrapPayload payload = new BootstrapPayload();
    payload.currentUser = Strings.hasText(currentUserID) ? findUser(currentUserID).orElse(null) : users.stream().findFirst().orElse(null);
    payload.stats = stats();
    payload.feed = mine ? feedMine(limit, currentUserID) : feed(limit);
    payload.users = listUsers();
    payload.media = listMedia(limit);
    payload.conversations = listConversations(limit, currentUserID);
    payload.instances = listInstances();
    return payload;
  }

  synchronized List<FederationInstance> listInstances() {
    return liveInstances();
  }

  synchronized List<SocialUser> listUsers() {
    return new ArrayList<>(users);
  }

  synchronized SocialUser createUser(CreateUserRequest req) {
    if (!Strings.hasText(req.handle)) {
      throw ApiException.badRequest("SOCIAL_HANDLE_REQUIRED", "validation", "handle is required");
    }
    SocialUser user = new SocialUser();
    user.id = nextID("user");
    user.handle = normalizeHandle(req.handle);
    user.displayName = Strings.or(req.displayName, user.handle.replaceFirst("^@", ""));
    user.bio = Strings.value(req.bio);
    user.instance = Strings.or(req.instance, "摩尔1号");
    user.wallet = Strings.value(req.wallet);
    user.avatarUrl = Strings.value(req.avatarUrl);
    user.fields = new ArrayList<>();
    user.featuredTags = new ArrayList<>();
    user.createdAt = Instant.now().toString();
    users.add(0, user);
    persistence.saveUser(user);
    persistence.saveSocialSnapshot(snapshot());
    return user;
  }

  synchronized SocialUser getUser(String id) {
    return findUser(id).orElseThrow(() -> ApiException.notFound("SOCIAL_USER_NOT_FOUND", "not_found", "user not found: " + id));
  }

  synchronized SocialUser updateUser(String id, UpdateUserRequest req) {
    SocialUser user = getUser(id);
    if (req.displayName != null) user.displayName = req.displayName;
    if (req.bio != null) user.bio = req.bio;
    if (req.instance != null) user.instance = resolveUserInstance(req.instance);
    if (req.avatarUrl != null) user.avatarUrl = req.avatarUrl;
    if (req.fields != null) user.fields = req.fields;
    if (req.featuredTags != null) user.featuredTags = req.featuredTags;
    if (req.isBot != null) user.isBot = req.isBot;
    persistence.saveUser(user);
    persistence.saveSocialSnapshot(snapshot());
    return user;
  }

  synchronized void followUser(String followerID, String targetID) {
    if (Objects.equals(followerID, targetID)) {
      throw ApiException.badRequest("SOCIAL_CANNOT_FOLLOW_SELF", "validation", "cannot follow yourself");
    }
    getUser(followerID);
    getUser(targetID);
    follows.computeIfAbsent(followerID, k -> new HashSet<>()).add(targetID);
    refreshFollowCounts();
    persistence.saveFollow(followerID, targetID);
    for (SocialUser user : users) {
      persistence.saveUser(user);
    }
    persistence.saveSocialSnapshot(snapshot());
  }

  synchronized void unfollowUser(String followerID, String targetID) {
    Set<String> set = follows.get(followerID);
    if (set != null) {
      set.remove(targetID);
    }
    refreshFollowCounts();
    persistence.deleteFollow(followerID, targetID);
    for (SocialUser user : users) {
      persistence.saveUser(user);
    }
    persistence.saveSocialSnapshot(snapshot());
  }

  synchronized List<SocialPost> feed(int limit) {
    return slice(posts.stream().filter(p -> !Strings.hasText(p.parentPostId)).sorted(Comparator.comparing((SocialPost p) -> p.createdAt).reversed()).toList(), limit);
  }

  synchronized List<SocialPost> feedMine(int limit, String currentUserID) {
    if (!Strings.hasText(currentUserID)) {
      return List.of();
    }
    return slice(posts.stream().filter(p -> !Strings.hasText(p.parentPostId) && currentUserID.equals(p.authorId)).sorted(Comparator.comparing((SocialPost p) -> p.createdAt).reversed()).toList(), limit);
  }

  synchronized SocialPost createPost(CreatePostRequest req) {
    SocialUser author = getUser(req.authorId);
    SocialPost post = new SocialPost();
    post.id = nextID("post");
    post.authorId = author.id;
    post.authorHandle = author.handle;
    post.authorName = author.displayName;
    post.instance = resolvePostInstance(req.instance, author.instance);
    post.kind = Strings.or(req.kind, Strings.hasText(req.parentPostId) ? "reply" : "post");
    post.content = Strings.value(req.content).trim();
    post.visibility = Strings.or(req.visibility, "public");
    post.storageUri = Strings.or(req.storageUri, localStorageURI(req));
    post.attestationUri = Strings.value(req.attestationUri);
    post.tags = req.tags == null ? new ArrayList<>() : new ArrayList<>(req.tags);
    post.media = new ArrayList<>();
    if (req.mediaIds != null) {
      for (String mediaID : req.mediaIds) {
        findMedia(mediaID).ifPresent(asset -> post.media.add(PostMedia.from(asset)));
      }
    }
    post.type = Strings.or(req.type, "post");
    post.interaction = Strings.or(req.interaction, "anyone");
    post.parentPostId = Strings.value(req.parentPostId);
    post.rootPostId = Strings.hasText(req.rootPostId) ? req.rootPostId : post.parentPostId;
    post.replyDepth = Strings.hasText(post.parentPostId) ? 1 : 0;
    if (req.pollOptions != null && req.pollOptions.size() >= 2) {
      Poll poll = new Poll();
      poll.options = req.pollOptions.stream().filter(Strings::hasText).map(label -> new PollOption(label, 0)).toList();
      poll.multiple = req.pollMultiple;
      poll.expiresAt = Instant.now().plus(Duration.ofMinutes(req.pollExpiresIn > 0 ? req.pollExpiresIn : 1440)).toString();
      poll.voters = new ArrayList<>();
      post.poll = poll;
    }
    post.createdAt = Instant.now().toString();
    applyPostChainAsset(post);
    posts.add(0, post);
    refreshPostCounts();
    persistence.savePost(post);
    persistence.saveSocialSnapshot(snapshot());
    return post;
  }

  synchronized SocialPost getPost(String id) {
    return findPost(id).orElseThrow(() -> ApiException.notFound("SOCIAL_POST_NOT_FOUND", "not_found", "post not found: " + id));
  }

  private String resolvePostInstance(String requestedInstance, String fallbackInstance) {
    String resolved = Strings.or(requestedInstance, fallbackInstance);
    boolean exists = instances.stream().anyMatch(instance -> instance.name.equals(resolved));
    return exists ? resolved : fallbackInstance;
  }

  private String resolveUserInstance(String requestedInstance) {
    String resolved = Strings.value(requestedInstance).trim();
    if (!Strings.hasText(resolved)) {
      throw ApiException.badRequest("SOCIAL_INSTANCE_REQUIRED", "validation", "instance is required");
    }
    boolean exists = instances.stream().anyMatch(instance -> instance.name.equals(resolved));
    if (!exists) {
      throw ApiException.badRequest("SOCIAL_INSTANCE_UNKNOWN", "validation", "unknown instance: " + resolved);
    }
    return resolved;
  }

  synchronized PostThread getPostThread(String id, int limit) {
    SocialPost post = getPost(id);
    PostThread thread = new PostThread();
    thread.post = post;
    thread.ancestors = ancestors(post);
    thread.replies = listReplies(id, limit);
    return thread;
  }

  synchronized List<SocialPost> listReplies(String id, int limit) {
    getPost(id);
    return slice(posts.stream()
        .filter(p -> id.equals(p.parentPostId) || id.equals(p.rootPostId))
        .sorted(Comparator.comparing(p -> p.createdAt))
        .toList(), limit);
  }

  synchronized SocialPost votePoll(String postID, String userID, List<Integer> optionIndices) {
    SocialPost post = getPost(postID);
    if (post.poll == null) {
      throw ApiException.badRequest("SOCIAL_POLL_MISSING", "validation", "post has no poll");
    }
    if (post.poll.voters.contains(userID)) {
      throw ApiException.badRequest("SOCIAL_ALREADY_VOTED", "validation", "already voted");
    }
    if (optionIndices == null || optionIndices.isEmpty()) {
      throw ApiException.badRequest("SOCIAL_NO_OPTIONS_SELECTED", "validation", "no options selected");
    }
    if (!post.poll.multiple && optionIndices.size() > 1) {
      throw ApiException.badRequest("SOCIAL_SINGLE_CHOICE_ONLY", "validation", "single choice only");
    }
    for (Integer idx : optionIndices) {
      if (idx == null || idx < 0 || idx >= post.poll.options.size()) {
        throw ApiException.badRequest("SOCIAL_INVALID_OPTION", "validation", "invalid option index");
      }
    }
    for (Integer idx : optionIndices) {
      post.poll.options.get(idx).votes++;
    }
    post.poll.voters.add(userID);
    persistence.savePost(post);
    persistence.saveSocialSnapshot(snapshot());
    return post;
  }

  synchronized List<MediaAsset> listMedia(int limit) {
    return slice(media.stream().sorted(Comparator.comparing((MediaAsset m) -> m.createdAt).reversed()).toList(), limit);
  }

  synchronized MediaAsset createMedia(CreateMediaRequest req) {
    getUser(req.ownerId);
    MediaAsset asset = new MediaAsset();
    asset.id = nextID("media");
    asset.ownerId = req.ownerId;
    asset.name = Strings.or(req.name, "media");
    asset.kind = Strings.or(req.kind, "image");
    asset.url = Strings.value(req.url);
    asset.storageUri = Strings.value(req.storageUri);
    asset.cid = Strings.value(req.cid);
    asset.sizeBytes = req.sizeBytes;
    asset.status = Strings.or(req.status, "stored");
    asset.createdAt = Instant.now().toString();
    media.add(0, asset);
    persistence.saveMedia(asset);
    persistence.saveSocialSnapshot(snapshot());
    return asset;
  }

  synchronized List<Conversation> listConversations(int limit, String currentUserID) {
    return slice(conversations.stream()
        .filter(c -> !Strings.hasText(currentUserID) || c.participantIds.contains(currentUserID))
        .sorted(Comparator.comparing((Conversation c) -> c.updatedAt).reversed())
        .map(this::enrichConversation)
        .toList(), limit);
  }

  synchronized Conversation createConversation(String initiatorID, CreateConversationRequest req) {
    getUser(initiatorID);
    Set<String> participants = new HashSet<>();
    if (req.participantIds != null) participants.addAll(req.participantIds);
    participants.add(initiatorID);
    if (participants.size() != 2) {
      throw ApiException.badRequest("SOCIAL_CONVERSATION_PARTICIPANTS", "validation", "conversation requires exactly two participants");
    }
    participants.forEach(this::getUser);
    Conversation conversation = new Conversation();
    conversation.id = nextID("conv");
    conversation.title = Strings.or(req.title, "New Conversation");
    conversation.participantIds = participants.stream().sorted().toList();
    conversation.initiatorId = initiatorID;
    conversation.encrypted = req.encrypted;
    conversation.messages = new ArrayList<>();
    conversation.updatedAt = Instant.now().toString();
    applyConversationChainAsset(conversation);
    conversations.add(0, conversation);
    persistence.saveConversation(conversation);
    persistence.saveSocialSnapshot(snapshot());
    return enrichConversation(conversation);
  }

  synchronized Conversation getConversation(String id) {
    return conversations.stream().filter(c -> c.id.equals(id)).findFirst().map(this::enrichConversation)
        .orElseThrow(() -> ApiException.notFound("SOCIAL_CONVERSATION_NOT_FOUND", "not_found", "conversation not found: " + id));
  }

  synchronized Conversation addMessage(String conversationID, CreateMessageRequest req) {
    SocialUser sender = getUser(req.senderId);
    Conversation conversation = getConversation(conversationID);
    if (!conversation.participantIds.contains(sender.id)) {
      throw ApiException.badRequest("SOCIAL_NOT_PARTICIPANT", "validation", "sender is not a participant in this conversation");
    }
    ChatMessage message = new ChatMessage();
    message.id = nextID("msg");
    message.conversationId = conversationID;
    message.senderId = sender.id;
    message.senderHandle = sender.handle;
    message.body = Strings.value(req.body).trim();
    message.createdAt = Instant.now().toString();
    applyMessageChainAsset(message);
    conversation.messages.add(message);
    conversation.updatedAt = message.createdAt;
    persistence.saveConversation(conversation);
    persistence.saveSocialSnapshot(snapshot());
    return enrichConversation(conversation);
  }

  private Conversation enrichConversation(Conversation source) {
    Conversation item = new Conversation();
    item.id = source.id;
    item.title = source.title;
    item.participantIds = new ArrayList<>(source.participantIds);
    item.initiatorId = source.initiatorId;
    item.encrypted = source.encrypted;
    item.assetUri = source.assetUri;
    item.chainId = source.chainId;
    item.txHash = source.txHash;
    item.contractAddress = source.contractAddress;
    item.explorerUrl = source.explorerUrl;
    item.messages = new ArrayList<>(source.messages);
    item.updatedAt = source.updatedAt;

    List<String> route = item.participantIds.stream()
        .map(this::findUser)
        .flatMap(Optional::stream)
        .map(user -> user.instance)
        .filter(Strings::hasText)
        .distinct()
        .toList();
    item.crossInstance = route.size() > 1;
    item.federationRoute = String.join(" -> ", route);
    return item;
  }

  synchronized List<Map<String, Object>> distribution() {
    Map<String, Integer> counts = new LinkedHashMap<>();
    for (SocialUser user : users) {
      counts.merge(user.instance, 1, Integer::sum);
    }
    List<Map<String, Object>> result = new ArrayList<>();
    counts.forEach((instance, count) -> result.add(Map.of("instance", instance, "users", count)));
    return result;
  }

  private List<FederationInstance> liveInstances() {
    Map<String, Integer> onlineByInstance = activeSessions.countOnlineByInstance(users);
    List<FederationInstance> result = new ArrayList<>();
    for (FederationInstance item : instances) {
      int online = onlineByInstance.getOrDefault(item.name, 0);
      long postCount = posts.stream().filter(post -> item.name.equals(post.instance)).count();
      FederationInstance live = new FederationInstance(
          item.name,
          item.focus,
          online + " 人在线",
          measuredInstanceLatency(item.name),
          online > 0 || postCount > 0 ? "healthy" : "idle"
      );
      result.add(live);
    }
    return result;
  }

  private String measuredInstanceLatency(String instanceName) {
    String probeURL = instanceProbeURLs().get(instanceName);
    if (!Strings.hasText(probeURL)) return "未探测";
    try {
      long started = System.nanoTime();
      HttpURLConnection conn = (HttpURLConnection) URI.create(probeURL).toURL().openConnection();
      conn.setRequestMethod("HEAD");
      conn.setConnectTimeout(1200);
      conn.setReadTimeout(1200);
      conn.connect();
      int status = conn.getResponseCode();
      conn.disconnect();
      long elapsedMs = Math.max(1L, (System.nanoTime() - started) / 1_000_000L);
      return status > 0 ? elapsedMs + " ms" : "不可达";
    } catch (Exception err) {
      return "不可达";
    }
  }

  private Map<String, String> instanceProbeURLs() {
    Map<String, String> result = new HashMap<>();
    String raw = Env.get("INSTANCE_PROBE_URLS", "");
    for (String entry : raw.split("[,;]")) {
      int idx = entry.indexOf('=');
      if (idx <= 0) continue;
      String name = entry.substring(0, idx).trim();
      String url = entry.substring(idx + 1).trim();
      if (Strings.hasText(name) && Strings.hasText(url)) {
        result.put(name, url);
      }
    }
    return result;
  }

  private void ensureChainAssets() {
    posts.forEach(this::applyPostChainAsset);
    conversations.forEach(conversation -> {
      applyConversationChainAsset(conversation);
      conversation.messages.forEach(this::applyMessageChainAsset);
    });
  }

  private void applyPostChainAsset(SocialPost post) {
    if (!Strings.hasText(post.storageUri)) post.storageUri = "asset://molesociety/post/" + post.id;
    if (!Strings.hasText(post.attestationUri)) post.attestationUri = "attestation://molesociety/post/" + post.id;
    if (Strings.hasText(post.txHash)) return;
    ChainAsset asset = createChainAsset("post", post.id, post.authorId + "|" + post.content + "|" + post.storageUri);
    post.chainId = asset.chainId;
    post.txHash = asset.txHash;
    post.contractAddress = asset.contractAddress;
    post.explorerUrl = asset.explorerUrl;
  }

  private void applyConversationChainAsset(Conversation conversation) {
    if (!Strings.hasText(conversation.assetUri)) conversation.assetUri = "asset://molesociety/conversation/" + conversation.id;
    if (Strings.hasText(conversation.txHash)) return;
    ChainAsset asset = createChainAsset("conversation", conversation.id, conversation.title + "|" + String.join(",", conversation.participantIds));
    conversation.chainId = asset.chainId;
    conversation.txHash = asset.txHash;
    conversation.contractAddress = asset.contractAddress;
    conversation.explorerUrl = asset.explorerUrl;
  }

  private void applyMessageChainAsset(ChatMessage message) {
    if (!Strings.hasText(message.assetUri)) message.assetUri = "asset://molesociety/message/" + message.id;
    if (Strings.hasText(message.txHash)) return;
    ChainAsset asset = createChainAsset("message", message.id, message.conversationId + "|" + message.senderId + "|" + message.body);
    message.chainId = asset.chainId;
    message.txHash = asset.txHash;
    message.contractAddress = asset.contractAddress;
    message.explorerUrl = asset.explorerUrl;
  }

  private ChainAsset createChainAsset(String kind, String id, String payload) {
    String chainID = Long.toString(Env.longValue("CHAIN_ID", 10143L));
    String txHash = "0x" + sha256Hex(kind + "|" + id + "|" + chainID + "|" + payload);
    String contractAddress = Strings.value(System.getProperty("ASSET_CONTRACT_ADDRESS", System.getProperty("CONTRACT_ADDRESS", "")));
    String explorerBase = Strings.or(System.getProperty("CHAIN_EXPLORER_BASE"), Strings.or(System.getProperty("CHAIN_EXPLORER_URL"), "https://testnet.monadexplorer.com/tx/"));
    String separator = explorerBase.endsWith("/") ? "" : "/";
    return new ChainAsset(chainID, txHash, contractAddress, explorerBase + separator + txHash);
  }

  private String sha256Hex(String value) {
    try {
      MessageDigest digest = MessageDigest.getInstance("SHA-256");
      byte[] hash = digest.digest(Strings.value(value).getBytes(StandardCharsets.UTF_8));
      StringBuilder out = new StringBuilder(hash.length * 2);
      for (byte item : hash) out.append(String.format("%02x", item));
      return out.toString();
    } catch (Exception err) {
      throw new IllegalStateException("failed to create chain asset hash", err);
    }
  }

  private void seed() {
    Instant now = Instant.now();
    instances.add(new FederationInstance("摩尔1号", "创作者主权与链上身份", "12.4k", "43 ms", "healthy"));
    instances.add(new FederationInstance("摩尔2号", "阅读社群与数字馆藏", "8.9k", "51 ms", "healthy"));
    instances.add(new FederationInstance("摩尔3号", "跨实例消息转发", "3.1k", "37 ms", "healthy"));
    instances.add(new FederationInstance("摩尔4号", "媒体与永续资源镜像", "5.7k", "49 ms", "healthy"));

    users.add(seedUser("user_archive", "@archive", "Whale Archive", "为创作者提供永久内容归档与链上身份锚定。", "摩尔1号", "0xa18f...3c92", "https://picsum.photos/seed/archive/128/128", 1284, 312, now.minus(Duration.ofHours(48))));
    users.add(seedUser("user_librarian", "@librarian", "Node Librarian", "把书籍确权、媒体存储和去中心化社交连接在一起。", "摩尔2号", "0x78fe...12ab", "https://picsum.photos/seed/librarian/128/128", 932, 221, now.minus(Duration.ofHours(36))));
    users.add(seedUser("user_fedilab", "@fedilab", "Open Federation Lab", "探索 ActivityPub、实时会话和多实例协作。", "摩尔3号", "0x95bc...09ee", "https://picsum.photos/seed/fedilab/128/128", 1650, 415, now.minus(Duration.ofHours(24))));

    MediaAsset manifest = new MediaAsset("media_manifesto", "user_archive", "genesis-manifesto.png", "image", "https://picsum.photos/seed/manifesto/1200/800", "ar://7xv91manifesto", "bafybeih7f4manifesto", 2400000, "mirrored", now.minus(Duration.ofHours(20)).toString());
    MediaAsset space = new MediaAsset("media_space", "user_librarian", "weekly-space.mp4", "video", "https://picsum.photos/seed/space/1200/800", "ar://weekly-space", "ar://space-video", 84000000, "stored", now.minus(Duration.ofHours(12)).toString());
    media.add(manifest);
    media.add(space);

    posts.add(seedPost("post_archive", "user_archive", "@archive", "Whale Archive", "摩尔1号", "统一 posts、replies、media 后，社交层可以直接成为出版资产的上下文索引。", "ar://post-archive", "attestation://archive/genesis", List.of("链上身份", "永久内容"), List.of(), 38, 128, now.minus(Duration.ofMinutes(130))));
    SocialPost librarian = seedPost("post_librarian", "user_librarian", "@librarian", "Node Librarian", "摩尔2号", "新媒体上传已同步到 Arweave 与 IPFS 双存储层。只要内容哈希一致，前端、实例、检索器都能独立重建同一份帖子上下文。", "ar://post-librarian", "storage://arweave/S1NfXo2...8vdP", List.of("Arweave", "IPFS", "永久媒体"), List.of(PostMedia.from(manifest)), 21, 64, now.minus(Duration.ofMinutes(90)));
    posts.add(librarian);
    posts.add(seedPost("post_fedilab", "user_fedilab", "@fedilab", "Open Federation Lab", "摩尔3号", "接下来要把当前的 relay server 从“扫码 mint”升级为 ActivityPub + 媒体索引 + 实时会话网关，让不同实例之间的关注、转发和聊天真正互通。", "ar://post-fedilab", "relay://federation/upgrade-plan", List.of("ActivityPub", "实时聊天", "Spring Boot"), List.of(), 55, 102, now.minus(Duration.ofMinutes(45))));
    posts.add(seedReply("reply_archive_1", "user_librarian", "@librarian", "Node Librarian", "摩尔2号", "统一 posts 之后，评论不再是孤立记录，线程、引用和排序都能复用同一套内容基础设施。", "post_archive", now.minus(Duration.ofMinutes(105))));
    posts.add(seedReply("reply_archive_2", "user_fedilab", "@fedilab", "Open Federation Lab", "摩尔3号", "后面接 ActivityPub 的时候，也能直接把 reply 当作 Note 的一种关系分支来处理。", "post_archive", now.minus(Duration.ofMinutes(95))));

    Conversation conv = new Conversation();
    conv.id = "conv_curator";
    conv.title = "Archive Curator";
    conv.participantIds = List.of("user_archive", "user_librarian");
    conv.initiatorId = "user_archive";
    conv.encrypted = true;
    conv.updatedAt = now.minus(Duration.ofMinutes(15)).toString();
    conv.messages = new ArrayList<>();
    conv.messages.add(new ChatMessage("msg_1", conv.id, "user_archive", "@archive", "我们把帖子正文上链证明，媒体放 Arweave，前端就能跨实例恢复。", now.minus(Duration.ofMinutes(20)).toString()));
    conv.messages.add(new ChatMessage("msg_2", conv.id, "user_librarian", "@librarian", "对，聊天部分我先做会话原型，后续再接 Matrix 或 libp2p。", now.minus(Duration.ofMinutes(18)).toString()));
    conversations.add(conv);

    follows.put("user_archive", new HashSet<>(List.of("user_librarian", "user_fedilab")));
    follows.put("user_librarian", new HashSet<>(List.of("user_archive")));
    follows.put("user_fedilab", new HashSet<>(List.of("user_archive")));
    refreshPostCounts();
    refreshFollowCounts();
  }

  private void normalizeMoleInstances() {
    Map<String, String> names = new HashMap<>(Map.of(
        "vault.social", "摩尔1号",
        "readers.polkadot", "摩尔2号",
        "relay.zone", "摩尔3号",
        "storage.zone", "摩尔4号"
    ));
    List<FederationInstance> normalized = new ArrayList<>();
    int next = 1;
    for (FederationInstance item : instances) {
      String name = Strings.or(names.get(item.name), item.name);
      if (!Strings.hasText(name) || !name.startsWith("摩尔")) {
        name = "摩尔" + next + "号";
      }
      names.put(item.name, name);
      normalized.add(new FederationInstance(name, item.focus, item.members, item.latency, item.status));
      next++;
    }
    instances.clear();
    instances.addAll(normalized);
    for (SocialUser user : users) {
      user.instance = Strings.or(names.get(user.instance), user.instance);
    }
    for (SocialPost post : posts) {
      post.instance = Strings.or(names.get(post.instance), post.instance);
    }
  }

  private SocialUser seedUser(String id, String handle, String displayName, String bio, String instance, String wallet, String avatarUrl, int followers, int following, Instant createdAt) {
    SocialUser user = new SocialUser();
    user.id = id;
    user.handle = handle;
    user.displayName = displayName;
    user.bio = bio;
    user.instance = instance;
    user.wallet = wallet;
    user.avatarUrl = avatarUrl;
    user.fields = new ArrayList<>();
    user.featuredTags = new ArrayList<>();
    user.followers = followers;
    user.following = following;
    user.createdAt = createdAt.toString();
    return user;
  }

  private SocialPost seedPost(String id, String authorID, String handle, String name, String instance, String content, String storageURI, String attestationURI, List<String> tags, List<PostMedia> mediaItems, int boosts, int likes, Instant createdAt) {
    SocialPost post = new SocialPost();
    post.id = id;
    post.authorId = authorID;
    post.authorHandle = handle;
    post.authorName = name;
    post.instance = instance;
    post.kind = "post";
    post.content = content;
    post.visibility = "public";
    post.storageUri = storageURI;
    post.attestationUri = attestationURI;
    post.tags = new ArrayList<>(tags);
    post.media = new ArrayList<>(mediaItems);
    post.type = "post";
    post.interaction = "anyone";
    post.boosts = boosts;
    post.likes = likes;
    post.createdAt = createdAt.toString();
    return post;
  }

  private SocialPost seedReply(String id, String authorID, String handle, String name, String instance, String content, String parentID, Instant createdAt) {
    SocialPost post = seedPost(id, authorID, handle, name, instance, content, "ar://" + id, "attestation://" + id, List.of("回复"), List.of(), 0, 12, createdAt);
    post.kind = "reply";
    post.parentPostId = parentID;
    post.rootPostId = parentID;
    post.replyDepth = 1;
    return post;
  }

  private Optional<SocialUser> findUser(String id) {
    return users.stream().filter(u -> u.id.equals(id)).findFirst();
  }

  private Optional<MediaAsset> findMedia(String id) {
    return media.stream().filter(m -> m.id.equals(id)).findFirst();
  }

  private Optional<SocialPost> findPost(String id) {
    return posts.stream().filter(p -> p.id.equals(id)).findFirst();
  }

  private List<SocialPost> ancestors(SocialPost post) {
    List<SocialPost> result = new ArrayList<>();
    String parentID = post.parentPostId;
    while (Strings.hasText(parentID)) {
      Optional<SocialPost> parent = findPost(parentID);
      if (parent.isEmpty()) break;
      result.add(0, parent.get());
      parentID = parent.get().parentPostId;
    }
    return result;
  }

  private void refreshPostCounts() {
    for (SocialPost post : posts) {
      post.replies = (int) posts.stream().filter(p -> post.id.equals(p.parentPostId) || post.id.equals(p.rootPostId)).count();
    }
  }

  private void refreshFollowCounts() {
    for (SocialUser user : users) {
      user.following = follows.getOrDefault(user.id, Set.of()).size();
      user.followers = (int) follows.values().stream().filter(set -> set.contains(user.id)).count();
    }
  }

  private static <T> List<T> slice(List<T> items, int limit) {
    int size = Math.max(0, Math.min(limit <= 0 ? 20 : limit, Math.min(items.size(), 100)));
    return new ArrayList<>(items.subList(0, size));
  }

  private static String normalizeHandle(String handle) {
    String trimmed = handle.trim();
    return trimmed.startsWith("@") ? trimmed : "@" + trimmed;
  }

  private static String nextID(String prefix) {
    return prefix + "_" + System.nanoTime();
  }

  private static String localStorageURI(CreatePostRequest req) {
    return "local://post/" + Math.abs(Objects.hash(req.content, req.authorId, System.nanoTime()));
  }
}

class ApiResponse<T> {
  public boolean ok;
  public T data;
  public String error;
  public String code;
  public String type;

  static <T> ResponseEntity<ApiResponse<T>> ok(T data) {
    ApiResponse<T> res = new ApiResponse<>();
    res.ok = true;
    res.data = data;
    return ResponseEntity.ok(res);
  }

  static <T> ResponseEntity<ApiResponse<T>> created(T data) {
    ApiResponse<T> res = new ApiResponse<>();
    res.ok = true;
    res.data = data;
    return ResponseEntity.status(HttpStatus.CREATED).body(res);
  }

  static <T> ResponseEntity<ApiResponse<T>> error(HttpStatus status, String error, String code, String type) {
    ApiResponse<T> res = new ApiResponse<>();
    res.ok = false;
    res.error = error;
    res.code = code;
    res.type = type;
    return ResponseEntity.status(status).body(res);
  }
}

class ApiException extends RuntimeException {
  final HttpStatus status;
  final String code;
  final String type;

  ApiException(HttpStatus status, String code, String type, String message) {
    super(message);
    this.status = status;
    this.code = code;
    this.type = type;
  }

  <T> ResponseEntity<ApiResponse<T>> toResponse() {
    return ApiResponse.error(status, getMessage(), code, type);
  }

  static ApiException badRequest(String code, String type, String message) {
    return new ApiException(HttpStatus.BAD_REQUEST, code, type, message);
  }

  static ApiException unauthorized(String code, String type, String message) {
    return new ApiException(HttpStatus.UNAUTHORIZED, code, type, message);
  }

  static ApiException forbidden(String code, String type, String message) {
    return new ApiException(HttpStatus.FORBIDDEN, code, type, message);
  }

  static ApiException notFound(String code, String type, String message) {
    return new ApiException(HttpStatus.NOT_FOUND, code, type, message);
  }

  static ApiException conflict(String code, String type, String message) {
    return new ApiException(HttpStatus.CONFLICT, code, type, message);
  }
}

class Env {
  static String get(String key, String fallback) {
    String property = System.getProperty(key);
    if (Strings.hasText(property)) return property;
    String environment = System.getenv(key);
    return Strings.hasText(environment) ? environment : fallback;
  }

  static String current() {
    return Env.get("APP_ENV", Env.get("ENV", "development")).toLowerCase(Locale.ROOT);
  }

  static boolean cookieSecure() {
    return "none".equalsIgnoreCase(cookieSameSite()) || "true".equalsIgnoreCase(Env.get("COOKIE_SECURE", "false"));
  }

  static String cookieSameSite() {
    return Env.get("COOKIE_SAMESITE", "Lax");
  }

  static boolean truthy(String value) {
    String normalized = Strings.value(value).trim().toLowerCase(Locale.ROOT);
    return normalized.equals("1") || normalized.equals("true") || normalized.equals("yes") || normalized.equals("on");
  }

  static long longValue(String key, long fallback) {
    try {
      return Long.parseLong(Env.get(key, Long.toString(fallback)).trim());
    } catch (NumberFormatException err) {
      return fallback;
    }
  }
}

class Strings {
  static boolean hasText(String value) {
    return value != null && !value.trim().isEmpty();
  }

  static String value(String value) {
    return value == null ? "" : value;
  }

  static String or(String value, String fallback) {
    return hasText(value) ? value.trim() : fallback;
  }
}

class UserField {
  public String name;
  public String value;
}

class SocialUser {
  public String id;
  public String handle;
  public String displayName;
  public String bio;
  public String instance;
  public String wallet;
  public String avatarUrl;
  public List<UserField> fields = new ArrayList<>();
  public List<String> featuredTags = new ArrayList<>();
  public boolean isBot;
  public int followers;
  public int following;
  public String createdAt;
}

class MediaAsset {
  public String id;
  public String ownerId;
  public String name;
  public String kind;
  public String url;
  public String storageUri;
  public String cid;
  public long sizeBytes;
  public String status;
  public String createdAt;

  MediaAsset() {
  }

  MediaAsset(String id, String ownerId, String name, String kind, String url, String storageUri, String cid, long sizeBytes, String status, String createdAt) {
    this.id = id;
    this.ownerId = ownerId;
    this.name = name;
    this.kind = kind;
    this.url = url;
    this.storageUri = storageUri;
    this.cid = cid;
    this.sizeBytes = sizeBytes;
    this.status = status;
    this.createdAt = createdAt;
  }
}

class PostMedia {
  public String id;
  public String name;
  public String url;
  public String kind;
  public String storageUri;
  public String cid;

  static PostMedia from(MediaAsset asset) {
    PostMedia item = new PostMedia();
    item.id = asset.id;
    item.name = asset.name;
    item.url = asset.url;
    item.kind = asset.kind;
    item.storageUri = asset.storageUri;
    item.cid = asset.cid;
    return item;
  }
}

class PollOption {
  public String label;
  public int votes;

  PollOption() {
  }

  PollOption(String label, int votes) {
    this.label = label;
    this.votes = votes;
  }
}

class Poll {
  public List<PollOption> options = new ArrayList<>();
  public String expiresAt;
  public boolean multiple;
  public List<String> voters = new ArrayList<>();
}

class SocialPost {
  public String id;
  public String authorId;
  public String authorHandle;
  public String authorName;
  public String instance;
  public String kind;
  public String content;
  public String visibility;
  public String storageUri;
  public String attestationUri;
  public String chainId;
  public String txHash;
  public String contractAddress;
  public String explorerUrl;
  public List<String> tags = new ArrayList<>();
  public List<PostMedia> media = new ArrayList<>();
  public String parentPostId;
  public String rootPostId;
  public int replyDepth;
  public int replies;
  public int boosts;
  public int likes;
  public String type;
  public String interaction;
  public Poll poll;
  public String createdAt;
}

class PostThread {
  public SocialPost post;
  public List<SocialPost> ancestors = new ArrayList<>();
  public List<SocialPost> replies = new ArrayList<>();
}

class ChatMessage {
  public String id;
  public String conversationId;
  public String senderId;
  public String senderHandle;
  public String body;
  public String assetUri;
  public String chainId;
  public String txHash;
  public String contractAddress;
  public String explorerUrl;
  public String createdAt;

  ChatMessage() {
  }

  ChatMessage(String id, String conversationId, String senderId, String senderHandle, String body, String createdAt) {
    this.id = id;
    this.conversationId = conversationId;
    this.senderId = senderId;
    this.senderHandle = senderHandle;
    this.body = body;
    this.createdAt = createdAt;
  }
}

class Conversation {
  public String id;
  public String title;
  public List<String> participantIds = new ArrayList<>();
  public String initiatorId;
  public boolean encrypted;
  public String assetUri;
  public String chainId;
  public String txHash;
  public String contractAddress;
  public String explorerUrl;
  public boolean crossInstance;
  public String federationRoute;
  public List<ChatMessage> messages = new ArrayList<>();
  public String updatedAt;
}

class ChainAsset {
  final String chainId;
  final String txHash;
  final String contractAddress;
  final String explorerUrl;

  ChainAsset(String chainId, String txHash, String contractAddress, String explorerUrl) {
    this.chainId = chainId;
    this.txHash = txHash;
    this.contractAddress = contractAddress;
    this.explorerUrl = explorerUrl;
  }
}

class FederationInstance {
  public String name;
  public String focus;
  public String members;
  public String latency;
  public String status;

  FederationInstance() {
  }

  FederationInstance(String name, String focus, String members, String latency, String status) {
    this.name = name;
    this.focus = focus;
    this.members = members;
    this.latency = latency;
    this.status = status;
  }
}

class SocialStats {
  public int users;
  public int posts;
  public int mediaAssets;
  public int conversations;

  SocialStats(int users, int posts, int mediaAssets, int conversations) {
    this.users = users;
    this.posts = posts;
    this.mediaAssets = mediaAssets;
    this.conversations = conversations;
  }
}

class BootstrapPayload {
  public SocialUser currentUser;
  public SocialStats stats;
  public List<SocialPost> feed = new ArrayList<>();
  public List<SocialUser> users = new ArrayList<>();
  public List<MediaAsset> media = new ArrayList<>();
  public List<Conversation> conversations = new ArrayList<>();
  public List<FederationInstance> instances = new ArrayList<>();
}

class AuthChallenge {
  public String nonce;
  public String address;
  public String message;
  public long chainId;
  public String issuedAt;
  public String expiresAt;
}

class AuthSessionRecord {
  public String id;
  public String userId;
  public String address;
  public String createdAt;
  public String expiresAt;
}

class Account {
  public String id;
  public String username;
  public String email;
  public String passwordHash;
  public String wallet;
  public String userId;
  public String status;
  public String createdAt;
  public String updatedAt;
}

class AuthResult {
  public final SocialUser user;
  public final AuthSessionRecord session;

  AuthResult(SocialUser user, AuthSessionRecord session) {
    this.user = user;
    this.session = session;
  }
}

class AuthSessionResponse {
  public String id;
  public String handle;
  public String displayName;
  public String instance;
  public String bio;
  public String avatarUrl;
  public String wallet;
  public List<UserField> fields = new ArrayList<>();
  public List<String> featuredTags = new ArrayList<>();
  public boolean isBot;

  static AuthSessionResponse from(SocialUser user) {
    AuthSessionResponse res = new AuthSessionResponse();
    res.id = user.id;
    res.handle = user.handle;
    res.displayName = user.displayName;
    res.instance = user.instance;
    res.bio = user.bio;
    res.avatarUrl = user.avatarUrl;
    res.wallet = user.wallet;
    res.fields = user.fields;
    res.featuredTags = user.featuredTags;
    res.isBot = user.isBot;
    return res;
  }
}

class AuthChallengeRequest {
  public String address;
  public long chainId;
}

class VerifyWalletRequest {
  public String address;
  public String nonce;
  public String signature;
}

class PasswordLoginRequest {
  public String identifier;
  public String password;
}

class RegisterRequest {
  public String username;
  public String email;
  public String password;
  public String walletAddress;
  public boolean autoWallet;
  public long chainId;
  public String signature;
  public String nonce;
}

class BindChallengeRequest {
  public String walletAddress;
  public long chainId;
}

class CreateUserRequest {
  public String handle;
  public String displayName;
  public String bio;
  public String instance;
  public String wallet;
  public String avatarUrl;

  CreateUserRequest() {
  }

  CreateUserRequest(String handle, String displayName, String bio, String instance, String wallet, String avatarUrl) {
    this.handle = handle;
    this.displayName = displayName;
    this.bio = bio;
    this.instance = instance;
    this.wallet = wallet;
    this.avatarUrl = avatarUrl;
  }
}

class UpdateUserRequest {
  public String displayName;
  public String bio;
  public String instance;
  public String avatarUrl;
  public List<UserField> fields;
  public List<String> featuredTags;
  public Boolean isBot;
}

class CreateMediaRequest {
  public String ownerId;
  public String name;
  public String kind;
  public String url;
  public String storageUri;
  public String cid;
  public long sizeBytes;
  public String status;
}

class CreatePostRequest {
  public String authorId;
  public String instance;
  public String kind;
  public String content;
  public String visibility;
  public String type;
  public String interaction;
  public String storageUri;
  public String attestationUri;
  public List<String> tags;
  public List<String> mediaIds;
  public String parentPostId;
  public String rootPostId;
  public List<String> pollOptions;
  public int pollExpiresIn;
  public boolean pollMultiple;
}

class VotePollRequest {
  public List<Integer> optionIndices = new ArrayList<>();
}

class CreateConversationRequest {
  public String title;
  public List<String> participantIds = new ArrayList<>();
  public boolean encrypted;
}

class CreateMessageRequest {
  public String senderId;
  public String body;
}

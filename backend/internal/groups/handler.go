package groups

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"joules/internal/auth"
	"joules/internal/db/sqlc"
)

type Handler struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func NewHandler(q *sqlc.Queries, pool *pgxpool.Pool) *Handler {
	return &Handler{q: q, pool: pool}
}

type apiResponse struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	slog.Error("groups error", "status", status, "error", err)
	msg := err.Error()
	if status >= 500 {
		msg = "internal server error"
	}
	writeJSON(w, status, apiResponse{Error: msg})
}

func getUserID(r *http.Request) (string, error) {
	userID, ok := r.Context().Value(auth.ContextUserID).(string)
	if !ok {
		return "", fmt.Errorf("unauthorized")
	}
	return userID, nil
}

func generateInviteCode() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// coerceRole converts an interface{} from COALESCE to string safely.
func coerceRole(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// Response types
type GroupItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	InviteCode  string `json:"invite_code"`
	MemberCount int32  `json:"member_count"`
	MyRole      string `json:"my_role"`
}

type PublicGroupItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MemberCount int32  `json:"member_count"`
}

type LeaderboardEntry struct {
	Rank       int    `json:"rank"`
	UserID     string `json:"user_id"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	Meals7d    int    `json:"meals_7d"`
	Calories7d int    `json:"calories_7d"`
	Points     int    `json:"points"`
}

type ChallengeProgress struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Value  int    `json:"value"`
}

type ChallengeItem struct {
	ID          string              `json:"id"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Metric      string              `json:"metric"`
	TargetValue int32               `json:"target_value"`
	StartDate   string              `json:"start_date"`
	EndDate     string              `json:"end_date"`
	Progress    []ChallengeProgress `json:"progress"`
}

func rowToGroupItem(g sqlc.GetGroupsByMemberRow) GroupItem {
	return GroupItem{
		ID:          g.ID,
		Name:        g.Name,
		Description: g.Description,
		Type:        g.Type,
		InviteCode:  g.InviteCode,
		MemberCount: g.MemberCount,
		MyRole:      g.MyRole,
	}
}

func (h *Handler) ListMyGroups(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	rows, err := h.q.GetGroupsByMember(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("list groups: %w", err))
		return
	}
	items := make([]GroupItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, rowToGroupItem(row))
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: items})
}

func (h *Handler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Type        string `json:"type"` // "private" | "public"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}
	if req.Type != "public" {
		req.Type = "private"
	}

	g, err := h.q.CreateGroup(r.Context(), sqlc.CreateGroupParams{
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		InviteCode:  generateInviteCode(),
		CreatedBy:   userID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("create group: %w", err))
		return
	}

	// Creator becomes admin member
	err = h.q.AddGroupMember(r.Context(), sqlc.AddGroupMemberParams{
		GroupID: g.ID,
		UserID:  userID,
		Role:    "admin",
	})
	if err != nil {
		_ = h.q.DeleteGroup(r.Context(), sqlc.DeleteGroupParams{ID: g.ID, CreatedBy: userID})
		writeError(w, http.StatusInternalServerError, fmt.Errorf("create group member: %w", err))
		return
	}

	writeJSON(w, http.StatusCreated, apiResponse{Data: GroupItem{
		ID:          g.ID,
		Name:        g.Name,
		Description: g.Description,
		Type:        g.Type,
		InviteCode:  g.InviteCode,
		MemberCount: 1,
		MyRole:      "admin",
	}})
}

func (h *Handler) DiscoverGroups(w http.ResponseWriter, r *http.Request) {
	rows, err := h.q.GetPublicGroups(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("discover groups: %w", err))
		return
	}
	items := make([]PublicGroupItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, PublicGroupItem{
			ID:          row.ID,
			Name:        row.Name,
			Description: row.Description,
			MemberCount: row.MemberCount,
		})
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: items})
}

func (h *Handler) JoinGroup(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	var req struct {
		InviteCode string `json:"invite_code"`
		GroupID    string `json:"group_id"` // for public groups
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var groupID string

	if req.InviteCode != "" {
		g, err := h.q.GetGroupByInviteCode(r.Context(), req.InviteCode)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(w, http.StatusNotFound, errors.New("invalid invite code"))
				return
			}
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		groupID = g.ID
	} else if req.GroupID != "" {
		groupID = req.GroupID
	} else {
		writeError(w, http.StatusBadRequest, errors.New("invite_code or group_id required"))
		return
	}

	err = h.q.AddGroupMember(r.Context(), sqlc.AddGroupMemberParams{
		GroupID: groupID,
		UserID:  userID,
		Role:    "member",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("join group: %w", err))
		return
	}

	row, err := h.q.GetGroupByID(r.Context(), sqlc.GetGroupByIDParams{ID: groupID, UserID: userID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: GroupItem{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		Type:        row.Type,
		InviteCode:  row.InviteCode,
		MemberCount: row.MemberCount,
		MyRole:      coerceRole(row.MyRole),
	}})
}

func (h *Handler) GetGroup(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	groupID := chi.URLParam(r, "id")

	row, err := h.q.GetGroupByID(r.Context(), sqlc.GetGroupByIDParams{ID: groupID, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, errors.New("group not found"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: GroupItem{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		Type:        row.Type,
		InviteCode:  row.InviteCode,
		MemberCount: row.MemberCount,
		MyRole:      coerceRole(row.MyRole),
	}})
}

func (h *Handler) LeaveGroup(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	groupID := chi.URLParam(r, "id")
	if err := h.q.RemoveGroupMember(r.Context(), sqlc.RemoveGroupMemberParams{GroupID: groupID, UserID: userID}); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("leave group: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]bool{"left": true}})
}

func (h *Handler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	groupID := chi.URLParam(r, "id")
	if err := h.q.DeleteGroup(r.Context(), sqlc.DeleteGroupParams{ID: groupID, CreatedBy: userID}); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("delete group: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]bool{"deleted": true}})
}

func (h *Handler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	groupID := chi.URLParam(r, "id")

	role, err := h.q.GetUserGroupRole(r.Context(), sqlc.GetUserGroupRoleParams{GroupID: groupID, UserID: userID})
	if err != nil || role == "" {
		writeError(w, http.StatusForbidden, errors.New("you are not a member of this group"))
		return
	}

	ctx := r.Context()

	rows, err := h.pool.Query(ctx, `
		SELECT
			up.user_id,
			up.name,
			gm.role,
			COUNT(DISTINCT m.id)::int AS meals_7d,
			COALESCE(SUM(fi.calories), 0)::int AS calories_7d,
			COALESCE(us.total_points, 0)::int AS points
		FROM group_members gm
		JOIN user_profiles up ON up.user_id = gm.user_id
		LEFT JOIN meals m ON m.user_id = gm.user_id
			AND m.timestamp >= CURRENT_TIMESTAMP - INTERVAL '7 days'
		LEFT JOIN food_items fi ON fi.meal_id = m.id
		LEFT JOIN user_stats us ON us.user_id = gm.user_id
		WHERE gm.group_id = $1
		GROUP BY up.user_id, up.name, gm.role, us.total_points
		ORDER BY points DESC, meals_7d DESC
	`, groupID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("leaderboard query: %w", err))
		return
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	rank := 1
	for rows.Next() {
		var e LeaderboardEntry
		if err := rows.Scan(&e.UserID, &e.Name, &e.Role, &e.Meals7d, &e.Calories7d, &e.Points); err != nil {
			continue
		}
		e.Rank = rank
		rank++
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []LeaderboardEntry{}
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: entries})
}

func (h *Handler) ListChallenges(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	groupID := chi.URLParam(r, "id")

	role, err := h.q.GetUserGroupRole(r.Context(), sqlc.GetUserGroupRoleParams{GroupID: groupID, UserID: userID})
	if err != nil || role == "" {
		writeError(w, http.StatusForbidden, errors.New("you are not a member of this group"))
		return
	}

	ctx := r.Context()

	challenges, err := h.q.GetGroupChallenges(ctx, groupID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("list challenges: %w", err))
		return
	}

	items := make([]ChallengeItem, 0, len(challenges))
	for _, c := range challenges {
		progress := h.getChallengeProgress(ctx, groupID, c)
		items = append(items, ChallengeItem{
			ID:          c.ID,
			Title:       c.Title,
			Description: c.Description,
			Metric:      c.Metric,
			TargetValue: c.TargetValue,
			StartDate:   c.StartDate.Format("2006-01-02"),
			EndDate:     c.EndDate.Format("2006-01-02"),
			Progress:    progress,
		})
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: items})
}

func (h *Handler) getChallengeProgress(ctx context.Context, groupID string, c sqlc.GroupChallenge) []ChallengeProgress {
	var query string
	switch c.Metric {
	case "meals":
		query = `
			SELECT up.user_id, up.name, COUNT(DISTINCT m.timestamp::date)::int AS value
			FROM group_members gm
			JOIN user_profiles up ON up.user_id = gm.user_id
			LEFT JOIN meals m ON m.user_id = gm.user_id
				AND m.timestamp::date BETWEEN $2 AND $3
			WHERE gm.group_id = $1
			GROUP BY up.user_id, up.name
			ORDER BY value DESC`
	case "calories":
		query = `
			SELECT up.user_id, up.name, COALESCE(SUM(fi.calories), 0)::int AS value
			FROM group_members gm
			JOIN user_profiles up ON up.user_id = gm.user_id
			LEFT JOIN meals m ON m.user_id = gm.user_id
				AND m.timestamp::date BETWEEN $2 AND $3
			LEFT JOIN food_items fi ON fi.meal_id = m.id
			WHERE gm.group_id = $1
			GROUP BY up.user_id, up.name
			ORDER BY value DESC`
	case "steps":
		query = `
			SELECT up.user_id, up.name, COALESCE(SUM(sl.step_count), 0)::int AS value
			FROM group_members gm
			JOIN user_profiles up ON up.user_id = gm.user_id
			LEFT JOIN step_logs sl ON sl.user_id = gm.user_id
				AND sl.date BETWEEN $2 AND $3
			WHERE gm.group_id = $1
			GROUP BY up.user_id, up.name
			ORDER BY value DESC`
	case "protein":
		query = `
			SELECT up.user_id, up.name, COALESCE(SUM(fi.protein_g), 0)::int AS value
			FROM group_members gm
			JOIN user_profiles up ON up.user_id = gm.user_id
			LEFT JOIN meals m ON m.user_id = gm.user_id
				AND m.timestamp::date BETWEEN $2 AND $3
			LEFT JOIN food_items fi ON fi.meal_id = m.id
			WHERE gm.group_id = $1
			GROUP BY up.user_id, up.name
			ORDER BY value DESC`
	default:
		return []ChallengeProgress{}
	}

	rows, err := h.pool.Query(ctx, query, groupID, c.StartDate, c.EndDate)
	if err != nil {
		return []ChallengeProgress{}
	}
	defer rows.Close()

	var result []ChallengeProgress
	for rows.Next() {
		var p ChallengeProgress
		if err := rows.Scan(&p.UserID, &p.Name, &p.Value); err != nil {
			continue
		}
		result = append(result, p)
	}
	if result == nil {
		result = []ChallengeProgress{}
	}
	return result
}

func (h *Handler) CreateChallenge(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	groupID := chi.URLParam(r, "id")

	// Only admin can create challenges
	role, err := h.q.GetUserGroupRole(r.Context(), sqlc.GetUserGroupRoleParams{GroupID: groupID, UserID: userID})
	if err != nil || role != "admin" {
		writeError(w, http.StatusForbidden, errors.New("only group admins can create challenges"))
		return
	}

	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Metric      string `json:"metric"`
		TargetValue int32  `json:"target_value"`
		StartDate   string `json:"start_date"`
		EndDate     string `json:"end_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	start, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid start_date: %w", err))
		return
	}
	end, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid end_date: %w", err))
		return
	}

	c, err := h.q.CreateChallenge(r.Context(), sqlc.CreateChallengeParams{
		GroupID:     groupID,
		Title:       req.Title,
		Description: req.Description,
		Metric:      req.Metric,
		TargetValue: req.TargetValue,
		StartDate:   start,
		EndDate:     end,
		CreatedBy:   userID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("create challenge: %w", err))
		return
	}

	writeJSON(w, http.StatusCreated, apiResponse{Data: ChallengeItem{
		ID:          c.ID,
		Title:       c.Title,
		Description: c.Description,
		Metric:      c.Metric,
		TargetValue: c.TargetValue,
		StartDate:   c.StartDate.Format("2006-01-02"),
		EndDate:     c.EndDate.Format("2006-01-02"),
		Progress:    []ChallengeProgress{},
	}})
}

-- name: CreateGroup :one
INSERT INTO groups (name, description, type, invite_code, created_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetGroupByID :one
SELECT
    g.*,
    (SELECT COUNT(*) FROM group_members WHERE group_id = g.id)::int AS member_count,
    COALESCE((SELECT role FROM group_members gm2 WHERE gm2.group_id = g.id AND gm2.user_id = $2), '') AS my_role
FROM groups g
WHERE g.id = $1;

-- name: GetGroupByInviteCode :one
SELECT * FROM groups WHERE invite_code = $1;

-- name: GetGroupsByMember :many
SELECT
    g.*,
    (SELECT COUNT(*) FROM group_members WHERE group_id = g.id)::int AS member_count,
    gm.role AS my_role
FROM groups g
JOIN group_members gm ON gm.group_id = g.id AND gm.user_id = $1
ORDER BY g.created_at DESC;

-- name: GetPublicGroups :many
SELECT
    g.id, g.name, g.description,
    (SELECT COUNT(*) FROM group_members WHERE group_id = g.id)::int AS member_count
FROM groups g
WHERE g.type = 'public'
ORDER BY member_count DESC
LIMIT 20;

-- name: AddGroupMember :exec
INSERT INTO group_members (group_id, user_id, role)
VALUES ($1, $2, $3)
ON CONFLICT (group_id, user_id) DO NOTHING;

-- name: RemoveGroupMember :exec
DELETE FROM group_members WHERE group_id = $1 AND user_id = $2;

-- name: IsGroupMember :one
SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2) AS is_member;

-- name: GetUserGroupRole :one
SELECT role FROM group_members WHERE group_id = $1 AND user_id = $2;

-- name: DeleteGroup :exec
DELETE FROM groups WHERE id = $1 AND created_by = $2;

-- name: CreateChallenge :one
INSERT INTO group_challenges (group_id, title, description, metric, target_value, start_date, end_date, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetGroupChallenges :many
SELECT * FROM group_challenges WHERE group_id = $1 ORDER BY created_at DESC;

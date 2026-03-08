import { useEffect, useState } from "react";
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  FormControlLabel,
  Paper,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
  Select,
  MenuItem,
  type SelectChangeEvent,
} from "@mui/material";
import {
  listAdminUsers,
  updateAdminUser,
  getAdminStats,
  type AdminUserView,
  type AdminStats,
  type UpdateUserRequest,
} from "../services/api.admin";

export function AdminPanel() {
  const [users, setUsers] = useState<AdminUserView[]>([]);
  const [stats, setStats] = useState<AdminStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState<Record<string, boolean>>({});
  const [edits, setEdits] = useState<Record<string, Partial<UpdateUserRequest>>>({});

  useEffect(() => {
    Promise.all([listAdminUsers(), getAdminStats()])
      .then(([u, s]) => {
        setUsers(u);
        setStats(s);
      })
      .catch((e) => setError(String(e)))
      .finally(() => setLoading(false));
  }, []);

  function getEdit(userId: string, user: AdminUserView): UpdateUserRequest {
    return {
      role: edits[userId]?.role ?? user.role,
      ai_enabled: edits[userId]?.ai_enabled ?? user.ai_enabled,
      token_limit: edits[userId]?.token_limit ?? user.token_limit,
      games_limit: edits[userId]?.games_limit ?? user.games_limit,
      notes: edits[userId]?.notes ?? user.notes ?? "",
    };
  }

  function patchEdit(userId: string, patch: Partial<UpdateUserRequest>) {
    setEdits((prev) => ({ ...prev, [userId]: { ...prev[userId], ...patch } }));
  }

  async function handleSave(user: AdminUserView) {
    const payload = getEdit(user.user_id, user);
    setSaving((prev) => ({ ...prev, [user.user_id]: true }));
    try {
      await updateAdminUser(user.user_id, payload);
      // Reflect the save locally
      setUsers((prev) =>
        prev.map((u) =>
          u.user_id === user.user_id ? { ...u, ...payload } : u,
        ),
      );
      setEdits((prev) => {
        const next = { ...prev };
        delete next[user.user_id];
        return next;
      });
    } catch (e) {
      setError(`Save failed for ${user.email || user.user_id}: ${String(e)}`);
    } finally {
      setSaving((prev) => ({ ...prev, [user.user_id]: false }));
    }
  }

  if (loading) {
    return (
      <Box sx={{ display: "flex", justifyContent: "center", mt: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3, maxWidth: 1200, mx: "auto" }}>
      <Typography variant="h4" sx={{ mb: 2, fontFamily: "Cinzel, serif" }}>
        Admin Panel
      </Typography>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {/* Stats bar */}
      {stats && (
        <Box sx={{ display: "flex", gap: 2, mb: 3, flexWrap: "wrap" }}>
          <Chip label={`${stats.total_users} users`} color="default" />
          <Chip label={`${stats.admin_users} admin`} color="warning" />
          <Chip label={`${stats.approved_users} approved`} color="success" />
          <Chip label={`${stats.restricted_users} restricted`} color="error" />
          <Chip
            label={`${stats.total_tokens_used.toLocaleString()} tokens used`}
            color="info"
          />
        </Box>
      )}

      <TableContainer component={Paper}>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>Email</TableCell>
              <TableCell>Role</TableCell>
              <TableCell>AI</TableCell>
              <TableCell>Token Limit</TableCell>
              <TableCell>Tokens Used</TableCell>
              <TableCell>Games Limit</TableCell>
              <TableCell>Notes</TableCell>
              <TableCell>Save</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {users.map((user) => {
              const e = getEdit(user.user_id, user);
              const dirty = user.user_id in edits;
              return (
                <TableRow
                  key={user.user_id}
                  sx={{ backgroundColor: dirty ? "action.hover" : undefined }}
                >
                  <TableCell>
                    <Typography variant="body2" sx={{ fontFamily: "monospace" }}>
                      {user.email || user.user_id.slice(0, 8) + "…"}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Select
                      size="small"
                      value={e.role}
                      onChange={(ev: SelectChangeEvent) =>
                        patchEdit(user.user_id, {
                          role: ev.target.value as UpdateUserRequest["role"],
                        })
                      }
                    >
                      <MenuItem value="admin">admin</MenuItem>
                      <MenuItem value="user">user</MenuItem>
                      <MenuItem value="restricted">restricted</MenuItem>
                    </Select>
                  </TableCell>
                  <TableCell>
                    <FormControlLabel
                      control={
                        <Switch
                          size="small"
                          checked={e.ai_enabled}
                          onChange={(ev) =>
                            patchEdit(user.user_id, {
                              ai_enabled: ev.target.checked,
                            })
                          }
                        />
                      }
                      label=""
                    />
                  </TableCell>
                  <TableCell>
                    <TextField
                      size="small"
                      type="number"
                      value={e.token_limit}
                      inputProps={{ min: 0, style: { width: 80 } }}
                      onChange={(ev) =>
                        patchEdit(user.user_id, {
                          token_limit: parseInt(ev.target.value, 10) || 0,
                        })
                      }
                    />
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2">
                      {user.tokens_used.toLocaleString()}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <TextField
                      size="small"
                      type="number"
                      value={e.games_limit}
                      inputProps={{ min: 0, style: { width: 60 } }}
                      onChange={(ev) =>
                        patchEdit(user.user_id, {
                          games_limit: parseInt(ev.target.value, 10) || 0,
                        })
                      }
                    />
                  </TableCell>
                  <TableCell>
                    <TextField
                      size="small"
                      value={e.notes}
                      inputProps={{ style: { width: 140 } }}
                      onChange={(ev) =>
                        patchEdit(user.user_id, { notes: ev.target.value })
                      }
                    />
                  </TableCell>
                  <TableCell>
                    <Button
                      size="small"
                      variant={dirty ? "contained" : "outlined"}
                      disabled={!dirty || saving[user.user_id]}
                      onClick={() => handleSave(user)}
                    >
                      {saving[user.user_id] ? (
                        <CircularProgress size={16} />
                      ) : (
                        "Save"
                      )}
                    </Button>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  );
}

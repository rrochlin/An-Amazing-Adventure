import {
   Button,
   Typography,
   Box,
   Divider,
   Paper,
   Tabs,
   Tab,
   Tooltip,
   Chip,
   LinearProgress,
   Alert,
   Snackbar,
} from '@mui/material';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import {
   type GameStateView,
   type ItemView,
   type RoomView,
} from '../types/types';
import type { CreateInviteResponse } from '../types/types';
import { useState } from 'react';
import { CreateInvite } from '../services/api.invites';

interface TabPanelProps {
   children?: React.ReactNode;
   index: number;
   value: number;
}

function TabPanel(props: TabPanelProps) {
   const { children, value, index, ...other } = props;
   return (
      <div
         role="tabpanel"
         hidden={value !== index}
         id={`game-tabpanel-${index}`}
         aria-labelledby={`game-tab-${index}`}
         {...other}
      >
         {value === index && <Box sx={{ py: 2 }}>{children}</Box>}
      </div>
   );
}

const SLOT_LABELS: Record<string, string> = {
   head: 'Head',
   chest: 'Chest',
   legs: 'Legs',
   hands: 'Hands',
   feet: 'Feet',
   back: 'Back',
};

const SLOT_ORDER = ['head', 'chest', 'legs', 'hands', 'feet', 'back'] as const;

interface GameInfoProps {
   gameState: GameStateView | null;
   sendAction: (subAction: string, payload: string) => void;
   /** When non-null, the Location tab displays this room instead of the current room (UI-FUT-4). */
   focusedRoom?: RoomView | null;
   /** Session UUID — used for invite generation */
   sessionId?: string;
   /** Whether the current user is the session owner */
   isOwner?: boolean;
}

export const GameInfo = ({
   gameState,
   sendAction,
   focusedRoom,
   sessionId,
   isOwner,
}: GameInfoProps) => {
   // UI-FUT-4: show focusedRoom when hovering/clicking map, otherwise fall back to current room
   const displayRoom = focusedRoom ?? gameState?.current_room;
   const currentRoom = displayRoom;
   const isFocusingOtherRoom =
      focusedRoom != null && focusedRoom.id !== gameState?.current_room?.id;
   const [tabValue, setTabValue] = useState(0);
   const [invite, setInvite] = useState<CreateInviteResponse | null>(null);
   const [inviteLoading, setInviteLoading] = useState(false);
   const [inviteError, setInviteError] = useState<string | null>(null);
   const [copyToast, setCopyToast] = useState(false);

   const handleGenerateInvite = async () => {
      if (!sessionId) return;
      setInviteLoading(true);
      setInviteError(null);
      try {
         const resp = await CreateInvite({
            session_id: sessionId,
            max_uses: 10,
            ttl_days: 7,
         });
         setInvite(resp);
      } catch {
         setInviteError('Failed to create invite');
      } finally {
         setInviteLoading(false);
      }
   };

   const handleCopyInvite = () => {
      if (!invite) return;
      navigator.clipboard.writeText(invite.url).then(() => setCopyToast(true));
   };

   const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
      setTabValue(newValue);
   };

   const SectionHeader = ({ children }: { children: React.ReactNode }) => (
      <Typography
         variant="h6"
         sx={{
            mb: 1,
            textTransform: 'uppercase',
            letterSpacing: '0.1em',
            fontSize: '0.9rem',
            borderLeft: '4px solid',
            borderColor: 'primary.main',
            pl: 1.5,
         }}
      >
         {children}
      </Typography>
   );

   const ItemButton = ({
      item,
      actions,
   }: {
      item: ItemView;
      actions: { label: string; onClick: () => void }[];
   }) => (
      <Box
         sx={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            mb: 0.5,
            gap: 1,
         }}
      >
         <Tooltip title={item.description || ''} placement="left" arrow>
            <Typography
               variant="body2"
               sx={{
                  fontFamily: 'Crimson Text, Georgia, serif',
                  fontSize: '1rem',
                  color: 'primary.main',
                  wordWrap: 'break-word',
                  overflowWrap: 'break-word',
                  flex: 1,
                  cursor: 'default',
               }}
            >
               {item.name}
               {item.equippable && item.slot && (
                  <Chip
                     label={item.slot}
                     size="small"
                     sx={{ ml: 0.5, fontSize: '0.6rem', height: '16px' }}
                  />
               )}
            </Typography>
         </Tooltip>
         <Box sx={{ display: 'flex', gap: 0.5, flexShrink: 0 }}>
            {actions.map((a) => (
               <Button
                  key={a.label}
                  size="small"
                  variant="outlined"
                  onClick={a.onClick}
                  sx={{
                     fontSize: '0.6rem',
                     py: 0,
                     px: 0.5,
                     minWidth: 0,
                     lineHeight: 1.4,
                     textTransform: 'capitalize',
                  }}
               >
                  {a.label}
               </Button>
            ))}
         </Box>
      </Box>
   );

   const equippedSlots = gameState?.player.equipment ?? {};

   return (
      <Box
         sx={{
            height: '100%',
            display: 'flex',
            flexDirection: 'column',
            minWidth: 0,
         }}
      >
         <Tabs
            value={tabValue}
            onChange={handleTabChange}
            sx={{
               borderBottom: 1,
               borderColor: 'divider',
               minHeight: '40px',
               minWidth: 0,
               '& .MuiTab-root': {
                  minHeight: '40px',
                  textTransform: 'uppercase',
                  fontSize: '0.7rem',
                  letterSpacing: '0.05em',
                  minWidth: 0,
                  px: 1,
               },
            }}
         >
            <Tab label="Location" />
            <Tab label="Inventory" />
            <Tab label="Equipment" />
            <Tab label="Room" />
            <Tab label="Party" />
         </Tabs>

         <Box
            sx={{
               flex: 1,
               overflow: 'auto',
               px: 2,
               minWidth: 0,
               '&::-webkit-scrollbar': { width: '8px' },
               '&::-webkit-scrollbar-track': {
                  background: 'background.default',
                  borderRadius: '4px',
               },
               '&::-webkit-scrollbar-thumb': {
                  background: 'primary.dark',
                  borderRadius: '4px',
               },
            }}
         >
            {/* Location Tab */}
            <TabPanel value={tabValue} index={0}>
               <Typography
                  variant="h6"
                  sx={{
                     mb: 0.5,
                     textAlign: 'center',
                     textTransform: 'uppercase',
                     letterSpacing: '0.1em',
                     borderBottom: '2px solid',
                     borderColor: isFocusingOtherRoom
                        ? 'info.main'
                        : 'primary.main',
                     pb: 1,
                     wordWrap: 'break-word',
                     overflowWrap: 'break-word',
                  }}
               >
                  {currentRoom?.name ?? 'Unknown Location'}
               </Typography>
               {isFocusingOtherRoom && (
                  <Typography
                     variant="caption"
                     sx={{
                        display: 'block',
                        textAlign: 'center',
                        color: 'info.main',
                        fontFamily: 'Crimson Text, Georgia, serif',
                        fontStyle: 'italic',
                        mb: 1,
                        fontSize: '0.75rem',
                     }}
                  >
                     Viewing from map — not your current location
                  </Typography>
               )}

               {/* Player health bar */}
               {gameState?.player && (
                  <Box sx={{ mb: 2 }}>
                     <Box
                        sx={{
                           display: 'flex',
                           justifyContent: 'space-between',
                           mb: 0.5,
                        }}
                     >
                        <Typography
                           variant="caption"
                           sx={{ color: 'text.secondary' }}
                        >
                           {gameState.player.name}
                        </Typography>
                        <Typography
                           variant="caption"
                           sx={{
                              color: gameState.player.alive
                                 ? 'success.main'
                                 : 'error.main',
                              fontWeight: 'bold',
                           }}
                        >
                           {gameState.player.alive
                              ? `${gameState.player.health}/100 HP`
                              : 'DEAD'}
                        </Typography>
                     </Box>
                     <LinearProgress
                        variant="determinate"
                        value={
                           gameState.player.alive ? gameState.player.health : 0
                        }
                        color={
                           gameState.player.health > 50
                              ? 'success'
                              : gameState.player.health > 20
                                ? 'warning'
                                : 'error'
                        }
                        sx={{ height: 6, borderRadius: 3 }}
                     />
                  </Box>
               )}

               <Paper
                  sx={(theme) => ({
                     p: 1.5,
                     mt: 1,
                     backgroundColor:
                        theme.palette.mode === 'dark'
                           ? 'rgba(201, 169, 98, 0.05)'
                           : 'rgba(160, 130, 109, 0.15)',
                     border:
                        theme.palette.mode === 'dark'
                           ? '1px solid rgba(201, 169, 98, 0.2)'
                           : '2px solid rgba(139, 111, 71, 0.4)',
                  })}
               >
                  <Typography
                     variant="body2"
                     sx={{
                        fontStyle: 'italic',
                        lineHeight: 1.6,
                        wordWrap: 'break-word',
                     }}
                  >
                     {currentRoom?.description || 'No description available'}
                  </Typography>
               </Paper>

               {/* Exits */}
               {currentRoom?.connections &&
                  Object.keys(currentRoom.connections).length > 0 && (
                     <Box sx={{ mt: 2 }}>
                        <SectionHeader>Exits</SectionHeader>
                        <Box
                           sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}
                        >
                           {Object.keys(currentRoom.connections).map((dir) => (
                              <Button
                                 key={dir}
                                 size="small"
                                 variant="outlined"
                                 onClick={() => sendAction('move', dir)}
                                 sx={{
                                    textTransform: 'capitalize',
                                    fontSize: '0.75rem',
                                    py: 0.25,
                                 }}
                              >
                                 {dir}
                              </Button>
                           ))}
                        </Box>
                     </Box>
                  )}
            </TabPanel>

            {/* Inventory Tab */}
            <TabPanel value={tabValue} index={1}>
               <SectionHeader>Your Pack</SectionHeader>
               <Divider
                  sx={(theme) => ({
                     mb: 1,
                     borderColor:
                        theme.palette.mode === 'dark'
                           ? 'rgba(201, 169, 98, 0.3)'
                           : 'rgba(139, 111, 71, 0.4)',
                  })}
               />
               {gameState?.player.inventory &&
               gameState.player.inventory.length > 0 ? (
                  <Box>
                     {gameState.player.inventory.map((item) => {
                        const isEquipped =
                           item.slot &&
                           equippedSlots[
                              item.slot as keyof typeof equippedSlots
                           ]?.id === item.id;
                        const actions: {
                           label: string;
                           onClick: () => void;
                        }[] = [
                           {
                              label: 'drop',
                              onClick: () => sendAction('drop', item.name),
                           },
                        ];
                        if (item.equippable && item.slot) {
                           if (isEquipped) {
                              actions.unshift({
                                 label: 'unequip',
                                 onClick: () =>
                                    sendAction('unequip', item.slot!),
                              });
                           } else {
                              actions.unshift({
                                 label: 'equip',
                                 onClick: () => sendAction('equip', item.name),
                              });
                           }
                        }
                        return (
                           <ItemButton
                              key={item.id}
                              item={item}
                              actions={actions}
                           />
                        );
                     })}
                  </Box>
               ) : (
                  <Typography
                     variant="body2"
                     sx={{ color: 'text.secondary', fontStyle: 'italic' }}
                  >
                     Your pack is empty
                  </Typography>
               )}
            </TabPanel>

            {/* Equipment Tab */}
            <TabPanel value={tabValue} index={2}>
               <SectionHeader>Equipped Gear</SectionHeader>
               <Divider
                  sx={(theme) => ({
                     mb: 2,
                     borderColor:
                        theme.palette.mode === 'dark'
                           ? 'rgba(201, 169, 98, 0.3)'
                           : 'rgba(139, 111, 71, 0.4)',
                  })}
               />
               <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                  {SLOT_ORDER.map((slot) => {
                     const equipped =
                        equippedSlots[slot as keyof typeof equippedSlots];
                     return (
                        <Box
                           key={slot}
                           sx={(theme) => ({
                              display: 'flex',
                              alignItems: 'center',
                              justifyContent: 'space-between',
                              p: 1,
                              borderRadius: 1,
                              border: '1px solid',
                              borderColor:
                                 theme.palette.mode === 'dark'
                                    ? equipped
                                       ? 'rgba(201, 169, 98, 0.5)'
                                       : 'rgba(255,255,255,0.1)'
                                    : equipped
                                      ? 'rgba(139, 111, 71, 0.6)'
                                      : 'rgba(0,0,0,0.12)',
                              backgroundColor:
                                 theme.palette.mode === 'dark'
                                    ? equipped
                                       ? 'rgba(201, 169, 98, 0.07)'
                                       : 'transparent'
                                    : equipped
                                      ? 'rgba(160, 130, 109, 0.1)'
                                      : 'transparent',
                           })}
                        >
                           <Box
                              sx={{
                                 display: 'flex',
                                 alignItems: 'center',
                                 gap: 1,
                                 flex: 1,
                                 minWidth: 0,
                              }}
                           >
                              <Typography
                                 variant="caption"
                                 sx={{
                                    textTransform: 'uppercase',
                                    letterSpacing: '0.08em',
                                    color: 'text.secondary',
                                    width: '44px',
                                    flexShrink: 0,
                                 }}
                              >
                                 {SLOT_LABELS[slot]}
                              </Typography>
                              <Tooltip
                                 title={equipped?.description ?? ''}
                                 placement="left"
                                 arrow
                              >
                                 <Typography
                                    variant="body2"
                                    sx={{
                                       fontFamily:
                                          'Crimson Text, Georgia, serif',
                                       color: equipped
                                          ? 'primary.main'
                                          : 'text.disabled',
                                       fontStyle: equipped
                                          ? 'normal'
                                          : 'italic',
                                       fontSize: '0.9rem',
                                       overflow: 'hidden',
                                       textOverflow: 'ellipsis',
                                       whiteSpace: 'nowrap',
                                    }}
                                 >
                                    {equipped ? equipped.name : '— empty —'}
                                 </Typography>
                              </Tooltip>
                           </Box>
                           {equipped && (
                              <Button
                                 size="small"
                                 variant="outlined"
                                 onClick={() => sendAction('unequip', slot)}
                                 sx={{
                                    fontSize: '0.6rem',
                                    py: 0,
                                    px: 0.5,
                                    minWidth: 0,
                                    lineHeight: 1.4,
                                    ml: 0.5,
                                    flexShrink: 0,
                                 }}
                              >
                                 remove
                              </Button>
                           )}
                        </Box>
                     );
                  })}
               </Box>
            </TabPanel>

            {/* Party Tab */}
            <TabPanel value={tabValue} index={4}>
               <SectionHeader>Party</SectionHeader>
               <Divider sx={{ mb: 1, borderColor: 'rgba(201,169,98,0.3)' }} />

               {/* Self */}
               {gameState?.self && (
                  <Box sx={{ mb: 2 }}>
                     <Box
                        sx={{
                           display: 'flex',
                           justifyContent: 'space-between',
                           mb: 0.5,
                        }}
                     >
                        <Typography
                           variant="caption"
                           sx={{ color: 'primary.main', fontWeight: 'bold' }}
                        >
                           {gameState.self.name} (you)
                        </Typography>
                        <Typography
                           variant="caption"
                           sx={{
                              color: gameState.self.alive
                                 ? 'success.main'
                                 : 'error.main',
                           }}
                        >
                           {gameState.self.alive
                              ? `${gameState.self.health} HP`
                              : 'DEAD'}
                        </Typography>
                     </Box>
                     <LinearProgress
                        variant="determinate"
                        value={gameState.self.alive ? gameState.self.health : 0}
                        color={
                           gameState.self.health > 50
                              ? 'success'
                              : gameState.self.health > 20
                                ? 'warning'
                                : 'error'
                        }
                        sx={{ height: 6, borderRadius: 3 }}
                     />
                  </Box>
               )}

               {/* Other party members */}
               {gameState?.party && gameState.party.length > 0 ? (
                  gameState.party.map((member) => (
                     <Box key={member.id} sx={{ mb: 2 }}>
                        <Box
                           sx={{
                              display: 'flex',
                              justifyContent: 'space-between',
                              mb: 0.5,
                           }}
                        >
                           <Typography
                              variant="caption"
                              sx={{ color: 'text.secondary' }}
                           >
                              {member.name}
                           </Typography>
                           <Typography
                              variant="caption"
                              sx={{
                                 color: member.alive
                                    ? 'success.main'
                                    : 'error.main',
                              }}
                           >
                              {member.alive ? `${member.health} HP` : 'DEAD'}
                           </Typography>
                        </Box>
                        <LinearProgress
                           variant="determinate"
                           value={member.alive ? member.health : 0}
                           color={
                              member.health > 50
                                 ? 'success'
                                 : member.health > 20
                                   ? 'warning'
                                   : 'error'
                           }
                           sx={{ height: 6, borderRadius: 3 }}
                        />
                     </Box>
                  ))
               ) : (
                  <Typography
                     variant="body2"
                     sx={{
                        color: 'text.secondary',
                        fontStyle: 'italic',
                        mb: 2,
                     }}
                  >
                     No other party members
                  </Typography>
               )}

               {/* Invite section — owner only */}
               {isOwner && sessionId && (
                  <Box sx={{ mt: 2 }}>
                     <SectionHeader>Invite</SectionHeader>
                     <Divider
                        sx={{ mb: 1, borderColor: 'rgba(201,169,98,0.3)' }}
                     />
                     {invite ? (
                        <Box>
                           <Typography
                              variant="body2"
                              sx={{
                                 fontFamily: 'monospace',
                                 mb: 1,
                                 wordBreak: 'break-all',
                              }}
                           >
                              Code: <strong>{invite.code}</strong>
                           </Typography>
                           <Box sx={{ display: 'flex', gap: 1 }}>
                              <Button
                                 size="small"
                                 variant="outlined"
                                 startIcon={
                                    <ContentCopyIcon fontSize="small" />
                                 }
                                 onClick={handleCopyInvite}
                              >
                                 Copy Link
                              </Button>
                              <Button
                                 size="small"
                                 variant="text"
                                 onClick={() => setInvite(null)}
                              >
                                 New
                              </Button>
                           </Box>
                        </Box>
                     ) : (
                        <Button
                           size="small"
                           variant="outlined"
                           onClick={handleGenerateInvite}
                           disabled={inviteLoading}
                        >
                           {inviteLoading
                              ? 'Generating...'
                              : 'Generate Invite Link'}
                        </Button>
                     )}
                     {inviteError && (
                        <Alert severity="error" sx={{ mt: 1 }}>
                           {inviteError}
                        </Alert>
                     )}
                  </Box>
               )}

               {/* Copy success toast */}
               <Snackbar
                  open={copyToast}
                  autoHideDuration={2500}
                  onClose={() => setCopyToast(false)}
                  message="Invite link copied!"
                  anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
               />
            </TabPanel>

            {/* Room Tab */}
            <TabPanel value={tabValue} index={3}>
               <SectionHeader>Items in Room</SectionHeader>
               <Divider
                  sx={(theme) => ({
                     mb: 1,
                     borderColor:
                        theme.palette.mode === 'dark'
                           ? 'rgba(201, 169, 98, 0.3)'
                           : 'rgba(139, 111, 71, 0.4)',
                  })}
               />
               {currentRoom?.items && currentRoom.items.length > 0 ? (
                  <Box>
                     {currentRoom.items.map((item) => (
                        <ItemButton
                           key={item.id}
                           item={item}
                           actions={[
                              {
                                 label: 'pick up',
                                 onClick: () =>
                                    sendAction('pick_up', item.name),
                              },
                           ]}
                        />
                     ))}
                  </Box>
               ) : (
                  <Typography
                     variant="body2"
                     sx={{ color: 'text.secondary', fontStyle: 'italic' }}
                  >
                     No items in this room
                  </Typography>
               )}

               <SectionHeader>Occupants</SectionHeader>
               <Divider
                  sx={(theme) => ({
                     mb: 1,
                     borderColor:
                        theme.palette.mode === 'dark'
                           ? 'rgba(201, 169, 98, 0.3)'
                           : 'rgba(139, 111, 71, 0.4)',
                  })}
               />
               {currentRoom?.occupants && currentRoom.occupants.length > 0 ? (
                  <Box component="ul" sx={{ pl: 2, m: 0 }}>
                     {currentRoom.occupants.map((occupant) => (
                        <Box component="li" key={occupant.id} sx={{ mb: 0.5 }}>
                           <Tooltip
                              title={occupant.description ?? ''}
                              placement="left"
                              arrow
                           >
                              <Typography
                                 variant="body2"
                                 sx={{
                                    fontFamily: 'Crimson Text, Georgia, serif',
                                    wordWrap: 'break-word',
                                    color: occupant.friendly
                                       ? 'text.primary'
                                       : 'error.main',
                                 }}
                              >
                                 {occupant.name}
                                 {!occupant.friendly && (
                                    <Chip
                                       label="hostile"
                                       size="small"
                                       color="error"
                                       sx={{
                                          ml: 0.5,
                                          fontSize: '0.6rem',
                                          height: '16px',
                                       }}
                                    />
                                 )}
                              </Typography>
                           </Tooltip>
                        </Box>
                     ))}
                  </Box>
               ) : (
                  <Typography
                     variant="body2"
                     sx={{ color: 'text.secondary', fontStyle: 'italic' }}
                  >
                     No occupants in this room
                  </Typography>
               )}
            </TabPanel>
         </Box>
      </Box>
   );
};

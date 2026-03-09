import { createFileRoute, redirect, useNavigate, useSearch } from "@tanstack/react-router";
import {
   Box,
   Button,
   Card,
   CardActionArea,
   CardContent,
   Chip,
   FormControl,
   FormHelperText,
   Grid,
   InputLabel,
   MenuItem,
   Paper,
   Select,
   Step,
   StepLabel,
   Stepper,
   TextField,
   Tooltip,
   Typography,
   Alert,
} from "@mui/material";
import { useState } from "react";
import { isAuthenticated } from "@/services/auth.service";
import { CreateGame, JoinCharacter } from "@/services/api.game";
import type { CharacterCreationData } from "@/types/types";
import { z } from "zod";

// ─── D&D Data ──────────────────────────────────────────────────────────────

const RACES = [
   { id: "human",      label: "Human",     desc: "Adaptable and ambitious. +1 to all ability scores.", subraces: [] },
   { id: "dwarf",      label: "Dwarf",     desc: "+2 CON. Stout and resilient underground warriors.", subraces: [
      { id: "hill-dwarf",     label: "Hill Dwarf",     desc: "+1 WIS, +1 HP per level" },
      { id: "mountain-dwarf", label: "Mountain Dwarf", desc: "+2 STR, medium armor proficiency" },
   ]},
   { id: "elf",        label: "Elf",       desc: "+2 DEX. Graceful, perceptive, immune to sleep magic.", subraces: [
      { id: "high-elf",  label: "High Elf",  desc: "+1 INT, one cantrip" },
      { id: "wood-elf",  label: "Wood Elf",  desc: "+1 WIS, 35ft speed, mask of the wild" },
   ]},
   { id: "halfling",   label: "Halfling",  desc: "+2 DEX. Lucky and nimble — reroll natural 1s.", subraces: [
      { id: "lightfoot-halfling", label: "Lightfoot", desc: "+1 CHA, naturally stealthy" },
      { id: "stout-halfling",     label: "Stout",     desc: "+1 CON, poison resistance" },
   ]},
   { id: "dragonborn", label: "Dragonborn", desc: "+2 STR, +1 CHA. Breath weapon and damage resistance.", subraces: [] },
   { id: "gnome",      label: "Gnome",     desc: "+2 INT. Cunning and inventive small folk.", subraces: [
      { id: "forest-gnome", label: "Forest Gnome", desc: "+1 DEX, minor illusion cantrip, speak with animals" },
      { id: "rock-gnome",   label: "Rock Gnome",   desc: "+1 CON, tinker proficiency" },
   ]},
   { id: "half-elf",   label: "Half-Elf",  desc: "+2 CHA, +1 to two others. Versatile and social.", subraces: [] },
   { id: "half-orc",   label: "Half-Orc",  desc: "+2 STR, +1 CON. Relentless endurance and savage attacks.", subraces: [] },
   { id: "tiefling",   label: "Tiefling",  desc: "+1 INT, +2 CHA. Infernal heritage grants fire resistance.", subraces: [] },
];

const CLASSES = [
   {
      id: "barbarian", label: "Barbarian", hitDie: "d12",
      desc: "Fierce warriors who enter a battle rage. Primary stat: STR.",
      features: ["Rage (bonus damage, resistance)", "Reckless Attack (advantage)"],
      skillCount: 2,
      skills: ["animal-handling", "athletics", "intimidation", "nature", "perception", "survival"],
      skillLabels: { "animal-handling": "Animal Handling", athletics: "Athletics", intimidation: "Intimidation", nature: "Nature", perception: "Perception", survival: "Survival" },
   },
   {
      id: "fighter", label: "Fighter", hitDie: "d10",
      desc: "Masters of combat with unmatched martial versatility. Primary stat: STR or DEX.",
      features: ["Second Wind (heal 1d10+level)", "Action Surge (extra action)"],
      skillCount: 2,
      skills: ["acrobatics", "animal-handling", "athletics", "history", "insight", "intimidation", "perception", "survival"],
      skillLabels: { acrobatics: "Acrobatics", "animal-handling": "Animal Handling", athletics: "Athletics", history: "History", insight: "Insight", intimidation: "Intimidation", perception: "Perception", survival: "Survival" },
   },
   {
      id: "monk", label: "Monk", hitDie: "d8",
      desc: "Masters of martial arts channeling mystical ki energy. Primary stat: DEX.",
      features: ["Flurry of Blows (2 unarmed strikes)", "Patient Defense (Dodge)", "Step of the Wind"],
      skillCount: 2,
      skills: ["acrobatics", "athletics", "history", "insight", "religion", "stealth"],
      skillLabels: { acrobatics: "Acrobatics", athletics: "Athletics", history: "History", insight: "Insight", religion: "Religion", stealth: "Stealth" },
   },
];

// Standard array values
const STANDARD_ARRAY = [15, 14, 13, 12, 10, 8];
const ABILITY_KEYS = ["str", "dex", "con", "int", "wis", "cha"] as const;
const ABILITY_LABELS: Record<string, string> = { str: "Strength", dex: "Dexterity", con: "Constitution", int: "Intelligence", wis: "Wisdom", cha: "Charisma" };

const PREFERENCE_OPTIONS = [
   { label: "Combat",      value: "combat" },
   { label: "Puzzles",     value: "puzzles" },
   { label: "Dialog",      value: "dialog" },
   { label: "Exploration", value: "exploration" },
   { label: "Chance",      value: "chance" },
   { label: "Stealth",     value: "stealth" },
   { label: "Crafting",    value: "crafting" },
   { label: "Mystery",     value: "mystery" },
];

// ─── Route ─────────────────────────────────────────────────────────────────

const CREATE_STEPS = ["Race & Class", "Ability Scores", "Skills", "Adventure"];
const JOIN_STEPS   = ["Race & Class", "Ability Scores", "Skills"];

export const Route = createFileRoute("/create")({
   component: CreateRoute,
   validateSearch: z.object({ session: z.string().optional() }),
   beforeLoad: () => {
      if (!isAuthenticated()) {
         throw redirect({ to: "/login", search: { redirect: location.href } });
      }
   },
});

// ─── Component ─────────────────────────────────────────────────────────────

function CreateRoute() {
   const navigate = useNavigate();
   const { session: joinSessionId } = useSearch({ from: "/create" });
   const isJoinPath = !!joinSessionId;
   const STEPS = isJoinPath ? JOIN_STEPS : CREATE_STEPS;

   const [step, setStep]           = useState(0);
   const [error, setError]         = useState<string | null>(null);
   const [isSubmitting, setIsSubmitting] = useState(false);

   // Step 0: Race & Class + name
   const [playerName, setPlayerName] = useState("");
   const [raceID, setRaceID]         = useState("");
   const [subraceID, setSubraceID]   = useState("");
   const [classID, setClassID]       = useState("");

   // Step 1: Ability scores (standard array assignment)
   const [abilityScores, setAbilityScores] = useState<Record<string, number>>({
      str: 10, dex: 10, con: 10, int: 10, wis: 10, cha: 10,
   });
   const [usedValues, setUsedValues] = useState<number[]>([]);

   // Step 2: Skills
   const [selectedSkills, setSelectedSkills] = useState<string[]>([]);

   // Step 3: Adventure preferences (create-only)
   const [preferences, setPreferences] = useState<string[]>([]);
   const [themeHint, setThemeHint]     = useState("");

   // ── Derived ──
   const selectedRace  = RACES.find((r) => r.id === raceID);
   const selectedClass = CLASSES.find((c) => c.id === classID);

   // ── Ability score helpers ──
   const assignValue = (ability: string, value: number) => {
      const oldValue = abilityScores[ability];
      const newUsed = usedValues.filter((v) => v !== oldValue);
      newUsed.push(value);
      setUsedValues(newUsed);
      setAbilityScores((prev) => ({ ...prev, [ability]: value }));
   };
   // availableValues used for display only (see "Available values" chips below)

   // ── Skill helpers ──
   const toggleSkill = (skill: string) => {
      setSelectedSkills((prev) => {
         if (prev.includes(skill)) return prev.filter((s) => s !== skill);
         if (selectedClass && prev.length >= selectedClass.skillCount) return prev;
         return [...prev, skill];
      });
   };

   const togglePreference = (value: string) => {
      setPreferences((prev) =>
         prev.includes(value) ? prev.filter((p) => p !== value) : [...prev, value],
      );
   };

   // ── Navigation ──
   const canAdvanceStep0 = !!raceID && !!classID && (!selectedRace?.subraces.length || !!subraceID) && !!playerName.trim();
   const canAdvanceStep1 = usedValues.length === 6;
   const canAdvanceStep2 = selectedClass ? selectedSkills.length === selectedClass.skillCount : false;

   const handleNext = () => {
      setStep((s) => s + 1);
      setError(null);
   };
   const handleBack = () => {
      setStep((s) => s - 1);
      setError(null);
   };

   // ── Submit ──
   const buildPayload = (): CharacterCreationData => ({
      name: playerName.trim(),
      race_id: raceID,
      subrace_id: subraceID || undefined,
      class_id: classID,
      ability_scores: abilityScores,
      selected_skills: selectedSkills,
      theme_hint: themeHint.trim() || undefined,
      preferences: preferences.length > 0 ? preferences : undefined,
   });

   const handleSubmit = async () => {
      setError(null);
      setIsSubmitting(true);
      try {
         const payload = buildPayload();
         if (isJoinPath) {
            await JoinCharacter(joinSessionId, payload);
            navigate({ to: "/game-{$sessionUUID}", params: { sessionUUID: joinSessionId } });
            return;
         }
         const result = await CreateGame(payload);
         navigate({ to: "/game-{$sessionUUID}", params: { sessionUUID: result.session_id } });
      } catch (e: unknown) {
         const msg = e instanceof Error ? e.message : "Failed to create adventure — please try again.";
         setError(msg);
         setIsSubmitting(false);
      }
   };

   // ── Render ──
   return (
      <Box sx={{ display: "flex", justifyContent: "center", alignItems: "flex-start", minHeight: "calc(100vh - 64px)", p: 4, pt: 6 }}>
         <Paper sx={{ maxWidth: 760, width: "100%", p: 4, backgroundImage: "linear-gradient(rgba(106, 78, 157, 0.05), rgba(201, 169, 98, 0.05))", border: "1px solid rgba(201, 169, 98, 0.2)" }}>
            <Typography variant="h3" sx={{ mb: 1, textAlign: "center", textTransform: "uppercase", letterSpacing: "0.1em", fontSize: "2rem", borderBottom: "3px solid", borderColor: "primary.main", pb: 2 }}>
               {isJoinPath ? "Join the Adventure" : "Forge Your Adventure"}
            </Typography>
            {isJoinPath && (
               <Alert severity="info" sx={{ mb: 2 }}>
                  You&apos;re joining an existing adventure. Create your character and you&apos;ll be dropped into the world.
               </Alert>
            )}

            <Stepper activeStep={step} sx={{ mb: 4, mt: 3 }}>
               {STEPS.map((label) => (
                  <Step key={label}><StepLabel>{label}</StepLabel></Step>
               ))}
            </Stepper>

            {/* ── Step 0: Race & Class ── */}
            {step === 0 && (
               <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
                  <TextField
                     label="Character Name"
                     value={playerName}
                     onChange={(e) => setPlayerName(e.target.value)}
                     fullWidth required
                     helperText="What are you called, adventurer?"
                     slotProps={{ htmlInput: { maxLength: 60, autoComplete: "off", name: "character-name-unique-field" } }}
                  />

                  {/* Race picker */}
                  <Box>
                     <Typography variant="subtitle2" sx={{ mb: 1.5, textTransform: "uppercase", letterSpacing: "0.08em" }}>
                        Choose Your Race
                     </Typography>
                     <Grid container spacing={1.5}>
                        {RACES.map((race) => (
                           <Grid size={{ xs: 6, sm: 4 }} key={race.id}>
                              <Tooltip title={race.desc} placement="top" arrow>
                                 <Card
                                    variant={raceID === race.id ? "outlined" : "elevation"}
                                    sx={{ borderColor: raceID === race.id ? "primary.main" : "transparent", borderWidth: 2, cursor: "pointer" }}
                                 >
                                    <CardActionArea onClick={() => { setRaceID(race.id); setSubraceID(""); }}>
                                       <CardContent sx={{ py: 1.5, px: 2, "&:last-child": { pb: 1.5 } }}>
                                          <Typography variant="body2" fontWeight={raceID === race.id ? 700 : 400}>{race.label}</Typography>
                                       </CardContent>
                                    </CardActionArea>
                                 </Card>
                              </Tooltip>
                           </Grid>
                        ))}
                     </Grid>
                  </Box>

                  {/* Subrace picker */}
                  {selectedRace && selectedRace.subraces.length > 0 && (
                     <FormControl fullWidth required>
                        <InputLabel>Subrace</InputLabel>
                        <Select value={subraceID} label="Subrace" onChange={(e) => setSubraceID(e.target.value)}>
                           {selectedRace.subraces.map((sr) => (
                              <MenuItem key={sr.id} value={sr.id}>
                                 <Box>
                                    <Typography variant="body2">{sr.label}</Typography>
                                    <Typography variant="caption" color="text.secondary">{sr.desc}</Typography>
                                 </Box>
                              </MenuItem>
                           ))}
                        </Select>
                     </FormControl>
                  )}

                  {/* Class picker */}
                  <Box>
                     <Typography variant="subtitle2" sx={{ mb: 1.5, textTransform: "uppercase", letterSpacing: "0.08em" }}>
                        Choose Your Class
                     </Typography>
                     <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
                        {CLASSES.map((cls) => (
                           <Card
                              key={cls.id}
                              variant={classID === cls.id ? "outlined" : "elevation"}
                              sx={{ borderColor: classID === cls.id ? "primary.main" : "transparent", borderWidth: 2, cursor: "pointer" }}
                           >
                              <CardActionArea onClick={() => { setClassID(cls.id); setSelectedSkills([]); }}>
                                 <CardContent sx={{ py: 1.5, px: 2, "&:last-child": { pb: 1.5 } }}>
                                    <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
                                       <Box>
                                          <Typography variant="body1" fontWeight={classID === cls.id ? 700 : 500}>
                                             {cls.label} <Chip label={cls.hitDie} size="small" sx={{ ml: 1, fontSize: "0.7rem" }} />
                                          </Typography>
                                          <Typography variant="caption" color="text.secondary">{cls.desc}</Typography>
                                       </Box>
                                    </Box>
                                    <Box sx={{ mt: 0.5, display: "flex", flexWrap: "wrap", gap: 0.5 }}>
                                       {cls.features.map((f) => (
                                          <Chip key={f} label={f} size="small" variant="outlined" sx={{ fontSize: "0.65rem" }} />
                                       ))}
                                    </Box>
                                 </CardContent>
                              </CardActionArea>
                           </Card>
                        ))}
                     </Box>
                  </Box>

                  <Box sx={{ display: "flex", justifyContent: "space-between", mt: 1 }}>
                     <Button variant="outlined" onClick={() => navigate({ to: "/" })}>Cancel</Button>
                     <Button variant="contained" onClick={handleNext} disabled={!canAdvanceStep0}>
                        Next: Ability Scores
                     </Button>
                  </Box>
               </Box>
            )}

            {/* ── Step 1: Ability Scores (Standard Array) ── */}
            {step === 1 && (
               <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                  <Typography variant="body2" sx={{ color: "text.secondary", fontStyle: "italic", fontFamily: '"Crimson Text", "Georgia", serif', fontSize: "1rem" }}>
                     Assign the standard array values to your ability scores: 15, 14, 13, 12, 10, 8. Each value can only be used once.
                  </Typography>

                  <Box sx={{ display: "flex", flexWrap: "wrap", gap: 1, mb: 1 }}>
                     <Typography variant="caption" sx={{ width: "100%", color: "text.secondary" }}>Available values:</Typography>
                     {STANDARD_ARRAY.map((v) => (
                        <Chip
                           key={v}
                           label={v}
                           variant={usedValues.includes(v) ? "filled" : "outlined"}
                           color={usedValues.includes(v) ? "default" : "primary"}
                           size="small"
                        />
                     ))}
                  </Box>

                  {ABILITY_KEYS.map((ab) => (
                     <Box key={ab} sx={{ display: "flex", alignItems: "center", gap: 2 }}>
                        <Typography variant="body2" sx={{ width: 100, fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.05em", flexShrink: 0 }}>
                           {ABILITY_LABELS[ab]}
                        </Typography>
                        <FormControl size="small" sx={{ minWidth: 100 }}>
                           <InputLabel>{ab.toUpperCase()}</InputLabel>
                           <Select
                              value={abilityScores[ab] === 10 && !usedValues.includes(10) && !STANDARD_ARRAY.includes(10) ? "" : abilityScores[ab]}
                              label={ab.toUpperCase()}
                              onChange={(e) => assignValue(ab, Number(e.target.value))}
                           >
                              <MenuItem value=""><em>—</em></MenuItem>
                              {STANDARD_ARRAY.map((v) => (
                                 <MenuItem key={v} value={v} disabled={usedValues.includes(v) && abilityScores[ab] !== v}>
                                    {v}
                                 </MenuItem>
                              ))}
                           </Select>
                        </FormControl>
                        <Typography variant="caption" color="text.secondary">
                           mod: {abilityScores[ab] >= 10 ? `+${Math.floor((abilityScores[ab] - 10) / 2)}` : Math.floor((abilityScores[ab] - 10) / 2)}
                        </Typography>
                     </Box>
                  ))}

                  <FormHelperText sx={{ color: canAdvanceStep1 ? "success.main" : "text.secondary" }}>
                     {usedValues.length}/6 values assigned
                  </FormHelperText>

                  <Box sx={{ display: "flex", justifyContent: "space-between", mt: 1 }}>
                     <Button variant="outlined" onClick={handleBack}>Back</Button>
                     <Button variant="contained" onClick={handleNext} disabled={!canAdvanceStep1}>
                        Next: Skills
                     </Button>
                  </Box>
               </Box>
            )}

            {/* ── Step 2: Skills ── */}
            {step === 2 && selectedClass && (
               <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                  <Typography variant="body2" sx={{ color: "text.secondary", fontStyle: "italic", fontFamily: '"Crimson Text", "Georgia", serif', fontSize: "1rem" }}>
                     Choose {selectedClass.skillCount} skill proficiencies from the {selectedClass.label} list.
                  </Typography>

                  <Box sx={{ display: "flex", flexWrap: "wrap", gap: 1 }}>
                        {selectedClass.skills.map((sk) => {
                        const isSelected = selectedSkills.includes(sk);
                        const isDisabled = !isSelected && selectedSkills.length >= selectedClass.skillCount;
                        const skillLabel = (selectedClass.skillLabels as unknown as Record<string, string>)[sk] ?? sk;
                        return (
                           <Chip
                              key={sk}
                              label={skillLabel}
                              clickable={!isDisabled}
                              onClick={() => !isDisabled && toggleSkill(sk)}
                              color={isSelected ? "primary" : "default"}
                              variant={isSelected ? "filled" : "outlined"}
                              sx={{ fontSize: "0.9rem", py: 0.5, opacity: isDisabled ? 0.4 : 1 }}
                           />
                        );
                     })}
                  </Box>

                  <FormHelperText sx={{ color: canAdvanceStep2 ? "success.main" : "text.secondary" }}>
                     {selectedSkills.length}/{selectedClass.skillCount} skills selected
                  </FormHelperText>

                  {error && <Alert severity="error">{error}</Alert>}

                  <Box sx={{ display: "flex", justifyContent: "space-between", mt: 1 }}>
                     <Button variant="outlined" onClick={handleBack}>Back</Button>
                     {isJoinPath ? (
                        <Button variant="contained" onClick={handleSubmit} disabled={!canAdvanceStep2 || isSubmitting} sx={{ minWidth: 160 }}>
                           {isSubmitting ? "Joining..." : "Enter the Adventure"}
                        </Button>
                     ) : (
                        <Button variant="contained" onClick={handleNext} disabled={!canAdvanceStep2}>
                           Next: Adventure
                        </Button>
                     )}
                  </Box>
               </Box>
            )}

            {/* ── Step 3: Adventure Preferences (create-only) ── */}
            {step === 3 && (
               <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
                  <Typography variant="body2" sx={{ color: "text.secondary", fontStyle: "italic", fontFamily: '"Crimson Text", "Georgia", serif', fontSize: "1rem" }}>
                     Shape the world you&apos;ll explore. All optional — the AI fills in the rest.
                  </Typography>

                  <Box>
                     <Typography variant="subtitle2" sx={{ mb: 1.5, textTransform: "uppercase", letterSpacing: "0.08em" }}>Preferred Gameplay</Typography>
                     <Box sx={{ display: "flex", flexWrap: "wrap", gap: 1 }}>
                        {PREFERENCE_OPTIONS.map((opt) => (
                           <Chip
                              key={opt.value}
                              label={opt.label}
                              clickable
                              onClick={() => togglePreference(opt.value)}
                              color={preferences.includes(opt.value) ? "primary" : "default"}
                              variant={preferences.includes(opt.value) ? "filled" : "outlined"}
                              sx={{ fontSize: "0.9rem", py: 0.5 }}
                           />
                        ))}
                     </Box>
                  </Box>

                  <TextField
                     label="World Tone / Theme Hint"
                     value={themeHint}
                     onChange={(e) => setThemeHint(e.target.value)}
                     fullWidth
                     helperText='e.g. "gritty noir", "high fantasy epic", "light-hearted comedy", "cosmic horror"'
                     inputProps={{ maxLength: 200 }}
                  />

                  {error && <Alert severity="error">{error}</Alert>}

                  <Box sx={{ display: "flex", justifyContent: "space-between", mt: 1 }}>
                     <Button variant="outlined" onClick={handleBack} disabled={isSubmitting}>Back</Button>
                     <Button variant="contained" onClick={handleSubmit} disabled={isSubmitting} sx={{ minWidth: 160 }}>
                        {isSubmitting ? "Creating..." : "Begin Adventure"}
                     </Button>
                  </Box>
               </Box>
            )}
         </Paper>
      </Box>
   );
}

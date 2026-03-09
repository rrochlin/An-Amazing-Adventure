import { createFileRoute, redirect, useNavigate, useSearch } from "@tanstack/react-router";
import {
   Alert,
   Box,
   Button,
   Card,
   CardActionArea,
   CardContent,
   Chip,
   Divider,
   FormControl,
   FormHelperText,
   InputLabel,
   MenuItem,
   Paper,
   Select,
   Step,
   StepLabel,
   Stepper,
   TextField,
   Typography,
} from "@mui/material";
import { useState } from "react";
import { isAuthenticated } from "@/services/auth.service";
import { CreateGame, JoinCharacter } from "@/services/api.game";
import type { CharacterCreationData } from "@/types/types";
import { z } from "zod";

// ─── D&D Static Data ────────────────────────────────────────────────────────

interface SubraceData {
   id: string;
   name: string;
   asiBonusLabel: string;
}

interface RaceCardData {
   id: string;
   name: string;
   asi: string;
   speed: string;
   traits: string[];
   subraces: SubraceData[];
}

const RACES: RaceCardData[] = [
   {
      id: "human", name: "Human",
      asi: "+1 to all ability scores", speed: "30 ft",
      traits: ["Extra Language", "Versatile"],
      subraces: [],
   },
   {
      id: "dwarf", name: "Dwarf",
      asi: "+2 CON", speed: "25 ft",
      traits: ["Darkvision", "Dwarven Resilience", "Stonecunning"],
      subraces: [
         { id: "hill_dwarf",     name: "Hill Dwarf",     asiBonusLabel: "+1 WIS, +1 HP/level" },
         { id: "mountain_dwarf", name: "Mountain Dwarf", asiBonusLabel: "+2 STR, medium armor" },
      ],
   },
   {
      id: "elf", name: "Elf",
      asi: "+2 DEX", speed: "30 ft",
      traits: ["Darkvision", "Fey Ancestry", "Trance"],
      subraces: [
         { id: "high_elf",  name: "High Elf",  asiBonusLabel: "+1 INT" },
         { id: "wood_elf",  name: "Wood Elf",  asiBonusLabel: "+1 WIS, 35 ft speed" },
      ],
   },
   {
      id: "halfling", name: "Halfling",
      asi: "+2 DEX", speed: "25 ft",
      traits: ["Lucky", "Brave", "Halfling Nimbleness"],
      subraces: [
         { id: "lightfoot", name: "Lightfoot", asiBonusLabel: "+1 CHA" },
         { id: "stout",     name: "Stout",     asiBonusLabel: "+1 CON, poison resistance" },
      ],
   },
   {
      id: "dragonborn", name: "Dragonborn",
      asi: "+2 STR, +1 CHA", speed: "30 ft",
      traits: ["Draconic Ancestry", "Breath Weapon", "Damage Resistance"],
      subraces: [],
   },
   {
      id: "gnome", name: "Gnome",
      asi: "+2 INT", speed: "25 ft",
      traits: ["Darkvision", "Gnome Cunning"],
      subraces: [
         { id: "forest_gnome", name: "Forest Gnome", asiBonusLabel: "+1 DEX" },
         { id: "rock_gnome",   name: "Rock Gnome",   asiBonusLabel: "+1 CON" },
      ],
   },
   {
      id: "half_elf", name: "Half-Elf",
      asi: "+2 CHA, +1 to two others", speed: "30 ft",
      traits: ["Darkvision", "Fey Ancestry", "Skill Versatility"],
      subraces: [],
   },
   {
      id: "half_orc", name: "Half-Orc",
      asi: "+2 STR, +1 CON", speed: "30 ft",
      traits: ["Darkvision", "Menacing", "Relentless Endurance", "Savage Attacks"],
      subraces: [],
   },
   {
      id: "tiefling", name: "Tiefling",
      asi: "+2 CHA, +1 INT", speed: "30 ft",
      traits: ["Darkvision", "Hellish Resistance", "Infernal Legacy"],
      subraces: [],
   },
];

interface ClassCardData {
   id: string;
   name: string;
   hitDie: string;
   primaryAbility: string;
   armorNote: string;
   flavor: string;
   features: string[];
   skillCount: number;
   skills: { id: string; label: string }[];
}

const CLASSES: ClassCardData[] = [
   {
      id: "barbarian", name: "Barbarian", hitDie: "d12",
      primaryAbility: "Strength",
      armorNote: "Light & Medium armor, Shields",
      flavor: "A fierce warrior driven by primal rage, the Barbarian shrugs off wounds that would fell lesser fighters.",
      features: [
         "Rage — Bonus damage, resistance to physical damage (2/day)",
         "Reckless Attack — Advantage on attacks (enemies also attack you with advantage)",
         "Extra Attack at level 5",
      ],
      skillCount: 2,
      skills: [
         { id: "animal_handling", label: "Animal Handling (WIS)" },
         { id: "athletics",       label: "Athletics (STR)" },
         { id: "intimidation",    label: "Intimidation (CHA)" },
         { id: "nature",          label: "Nature (INT)" },
         { id: "perception",      label: "Perception (WIS)" },
         { id: "survival",        label: "Survival (WIS)" },
      ],
   },
   {
      id: "fighter", name: "Fighter", hitDie: "d10",
      primaryAbility: "Strength or Dexterity",
      armorNote: "All armor, Shields",
      flavor: "Master of weapons and armor, the Fighter is a versatile warrior capable of multiple attacks and tactical recovery.",
      features: [
         "Second Wind — Heal 1d10+level HP as a bonus action (1/short rest)",
         "Action Surge — Take an extra action on your turn (1/short rest)",
         "Extra Attack at level 5",
      ],
      skillCount: 2,
      skills: [
         { id: "acrobatics",      label: "Acrobatics (DEX)" },
         { id: "animal_handling", label: "Animal Handling (WIS)" },
         { id: "athletics",       label: "Athletics (STR)" },
         { id: "history",         label: "History (INT)" },
         { id: "insight",         label: "Insight (WIS)" },
         { id: "intimidation",    label: "Intimidation (CHA)" },
         { id: "perception",      label: "Perception (WIS)" },
         { id: "survival",        label: "Survival (WIS)" },
      ],
   },
   {
      id: "monk", name: "Monk", hitDie: "d8",
      primaryAbility: "Dexterity & Wisdom",
      armorNote: "No armor (Unarmored Defense)",
      flavor: "A master of martial arts, the Monk harnesses ki energy to perform extraordinary feats of speed and power.",
      features: [
         "Flurry of Blows — 2 extra unarmed strikes (costs 1 Ki point)",
         "Patient Defense — Take Dodge action as bonus action (costs 1 Ki point)",
         "Step of the Wind — Dash or Disengage as bonus action (costs 1 Ki point)",
         "Extra Attack at level 5",
      ],
      skillCount: 2,
      skills: [
         { id: "acrobatics", label: "Acrobatics (DEX)" },
         { id: "athletics",  label: "Athletics (STR)" },
         { id: "history",    label: "History (INT)" },
         { id: "insight",    label: "Insight (WIS)" },
         { id: "religion",   label: "Religion (INT)" },
         { id: "stealth",    label: "Stealth (DEX)" },
      ],
   },
];

const STANDARD_ARRAY = [15, 14, 13, 12, 10, 8];
const ABILITY_KEYS = ["str", "dex", "con", "int", "wis", "cha"] as const;
type AbilityKey = (typeof ABILITY_KEYS)[number];
const ABILITY_LABELS: Record<AbilityKey, string> = {
   str: "Strength", dex: "Dexterity", con: "Constitution",
   int: "Intelligence", wis: "Wisdom", cha: "Charisma",
};

// Racial ASI bonuses keyed by race_id and optional subrace_id
const RACIAL_ASI: Record<string, Partial<Record<AbilityKey, number>>> = {
   human:      { str: 1, dex: 1, con: 1, int: 1, wis: 1, cha: 1 },
   dwarf:      { con: 2 },
   hill_dwarf: { wis: 1 },
   mountain_dwarf: { str: 2 },
   elf:        { dex: 2 },
   high_elf:   { int: 1 },
   wood_elf:   { wis: 1 },
   halfling:   { dex: 2 },
   lightfoot:  { cha: 1 },
   stout:      { con: 1 },
   dragonborn: { str: 2, cha: 1 },
   gnome:      { int: 2 },
   forest_gnome: { dex: 1 },
   rock_gnome:   { con: 1 },
   half_elf:   { cha: 2 },
   half_orc:   { str: 2, con: 1 },
   tiefling:   { cha: 2, int: 1 },
};

function getRacialBonuses(raceId: string, subraceId: string): Partial<Record<AbilityKey, number>> {
   const base = RACIAL_ASI[raceId] ?? {};
   const sub  = subraceId ? (RACIAL_ASI[subraceId] ?? {}) : {};
   const merged: Partial<Record<AbilityKey, number>> = { ...base };
   for (const k of ABILITY_KEYS) {
      if (sub[k]) merged[k] = (merged[k] ?? 0) + sub[k]!;
   }
   return merged;
}

function abilityModifier(score: number): string {
   const mod = Math.floor((score - 10) / 2);
   return mod >= 0 ? `+${mod}` : `${mod}`;
}

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

// ─── Steps ──────────────────────────────────────────────────────────────────

const CREATE_STEPS = ["Name", "Race", "Class", "Ability Scores", "Skills", "Adventure", "Review"];
const JOIN_STEPS   = ["Name", "Race", "Class", "Ability Scores", "Skills", "Review"];

// step indices for each mode
const STEP = {
   NAME: 0, RACE: 1, CLASS: 2, ABILITIES: 3, SKILLS: 4,
   ADVENTURE_OR_REVIEW: 5, // "Adventure" in create mode, "Review" in join mode
   REVIEW: 6,              // only in create mode
} as const;

// ─── Route ──────────────────────────────────────────────────────────────────

export const Route = createFileRoute("/create")({
   component: CreateRoute,
   validateSearch: z.object({ session: z.string().optional() }),
   beforeLoad: () => {
      if (!isAuthenticated()) {
         throw redirect({ to: "/login", search: { redirect: location.href } });
      }
   },
});

// ─── Component ──────────────────────────────────────────────────────────────

/** Exported for unit testing (bypasses TanStack route lazy wrapper). */
export function CreateRoute() {
   const navigate = useNavigate();
   const { session: joinSessionId } = useSearch({ from: "/create" });
   const isJoinMode = !!joinSessionId;
   const STEPS = isJoinMode ? JOIN_STEPS : CREATE_STEPS;

   const [step, setStep]                 = useState(0);
   const [error, setError]               = useState<string | null>(null);
   const [isSubmitting, setIsSubmitting] = useState(false);

   // Step 0 — Name + Backstory
   const [playerName, setPlayerName] = useState("");
   const [backstory, setBackstory]   = useState("");

   // Step 1 — Race
   const [raceID,    setRaceID]    = useState("");
   const [subraceID, setSubraceID] = useState("");

   // Step 2 — Class
   const [classID, setClassID] = useState("");

   // Step 3 — Ability Scores (standard array)
   const [abilityScores, setAbilityScores] = useState<Record<AbilityKey, number>>({
      str: 0, dex: 0, con: 0, int: 0, wis: 0, cha: 0,
   });
   const [usedValues, setUsedValues] = useState<number[]>([]);

   // Step 4 — Skills
   const [selectedSkills, setSelectedSkills] = useState<string[]>([]);

   // Step 5 — Adventure Preferences (create mode only)
   const [preferences, setPreferences] = useState<string[]>([]);
   const [themeHint,   setThemeHint]   = useState("");

   // ── Derived ──
   const selectedRace  = RACES.find((r) => r.id === raceID);
   const selectedClass = CLASSES.find((c) => c.id === classID);
   const racialBonuses = getRacialBonuses(raceID, subraceID);

   // Final scores (base + racial bonus) used in Review step
   const finalScores = ABILITY_KEYS.reduce<Record<AbilityKey, number>>((acc, ab) => {
      acc[ab] = (abilityScores[ab] || 0) + (racialBonuses[ab] ?? 0);
      return acc;
   }, { str: 0, dex: 0, con: 0, int: 0, wis: 0, cha: 0 });

   // HP at level 1: hit die max + CON mod
   function calcHP(): number {
      if (!selectedClass) return 0;
      const hitDieMax = { d12: 12, d10: 10, d8: 8 }[selectedClass.hitDie] ?? 8;
      const conMod    = Math.floor((finalScores.con - 10) / 2);
      return hitDieMax + conMod;
   }

   // AC at level 1: Monk uses 10 + DEX + WIS (unarmored defense), others use 10 + DEX
   function calcAC(): number {
      const dexMod = Math.floor((finalScores.dex - 10) / 2);
      if (classID === "monk") {
         const wisMod = Math.floor((finalScores.wis - 10) / 2);
         return 10 + dexMod + wisMod;
      }
      return 10 + dexMod;
   }

   // ── Ability score helpers ──
   const assignValue = (ability: AbilityKey, value: number) => {
      const oldValue = abilityScores[ability];
      setUsedValues((prev) => {
         const next = prev.filter((v) => v !== oldValue);
         if (value !== 0) next.push(value);
         return next;
      });
      setAbilityScores((prev) => ({ ...prev, [ability]: value }));
   };

   // ── Skill helpers ──
   const toggleSkill = (skillId: string) => {
      setSelectedSkills((prev) => {
         if (prev.includes(skillId)) return prev.filter((s) => s !== skillId);
         if (selectedClass && prev.length >= selectedClass.skillCount) return prev;
         return [...prev, skillId];
      });
   };

   const togglePreference = (value: string) => {
      setPreferences((prev) =>
         prev.includes(value) ? prev.filter((p) => p !== value) : [...prev, value],
      );
   };

   const handleNext = () => { setStep((s) => s + 1); setError(null); };
   const handleBack = () => { setStep((s) => s - 1); setError(null); };

   // ── Determine if current step is the last action step before submit ──
   // In join mode: step 5 is Review (no Adventure step)
   // In create mode: step 5 is Adventure, step 6 is Review
   const isReviewStep = isJoinMode ? step === 5 : step === 6;
   const isAdventureStep = !isJoinMode && step === 5;

   // ── Submit ──
   const buildPayload = (): CharacterCreationData => ({
      name:            playerName.trim(),
      backstory:       backstory.trim() || undefined,
      race_id:         raceID,
      subrace_id:      subraceID || undefined,
      class_id:        classID,
      ability_scores:  abilityScores,
      selected_skills: selectedSkills,
      theme_hint:      themeHint.trim() || undefined,
      preferences:     preferences.length > 0 ? preferences : undefined,
   });

   const handleSubmit = async () => {
      setError(null);
      setIsSubmitting(true);
      try {
         const payload = buildPayload();
         if (isJoinMode) {
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

   // ── Nav helpers ──
   const navButtons = (canNext: boolean, nextLabel?: string) => (
      <Box sx={{ display: "flex", justifyContent: "space-between", mt: 3 }}>
         <Button variant="outlined" onClick={step === 0 ? () => navigate({ to: "/" }) : handleBack}>
            {step === 0 ? "Cancel" : "Back"}
         </Button>
         <Button variant="contained" onClick={handleNext} disabled={!canNext}>
            {nextLabel ?? "Next"}
         </Button>
      </Box>
   );

   // ── Render ──
   return (
      <Box sx={{ display: "flex", justifyContent: "center", alignItems: "flex-start", minHeight: "calc(100vh - 64px)", p: { xs: 2, sm: 4 }, pt: 6 }}>
         <Paper sx={{ maxWidth: 800, width: "100%", p: { xs: 2, sm: 4 }, backgroundImage: "linear-gradient(rgba(106, 78, 157, 0.05), rgba(201, 169, 98, 0.05))", border: "1px solid rgba(201, 169, 98, 0.2)" }}>
            <Typography variant="h3" sx={{ mb: 1, textAlign: "center", textTransform: "uppercase", letterSpacing: "0.1em", fontSize: "2rem", borderBottom: "3px solid", borderColor: "primary.main", pb: 2 }}>
               {isJoinMode ? "Join the Adventure" : "Forge Your Adventure"}
            </Typography>
            {isJoinMode && (
               <Alert severity="info" sx={{ mb: 2 }}>
                  You&apos;re joining an existing adventure. Create your character and you&apos;ll be dropped into the world.
               </Alert>
            )}

            <Stepper activeStep={step} sx={{ mb: 4, mt: 3 }} alternativeLabel>
               {STEPS.map((label) => (
                  <Step key={label}><StepLabel>{label}</StepLabel></Step>
               ))}
            </Stepper>

            {/* ── Step 0: Name + Backstory ── */}
            {step === STEP.NAME && (
               <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
                  <TextField
                     label="Character Name"
                     value={playerName}
                     onChange={(e) => setPlayerName(e.target.value)}
                     fullWidth required
                     helperText="What are you called, adventurer? (2–40 characters)"
                     slotProps={{ htmlInput: { maxLength: 40, autoComplete: "off" } }}
                  />
                  <TextField
                     label="Backstory (optional)"
                     value={backstory}
                     onChange={(e) => setBackstory(e.target.value)}
                     fullWidth multiline rows={3}
                     helperText="A brief 2–3 sentence origin for your character. The Dungeon Master will weave it into the story."
                     slotProps={{ htmlInput: { maxLength: 300 } }}
                  />
                  {navButtons(playerName.trim().length >= 2, "Next: Race")}
               </Box>
            )}

            {/* ── Step 1: Race ── */}
            {step === STEP.RACE && (
               <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
                  <Typography variant="body2" sx={{ color: "text.secondary", fontStyle: "italic", fontFamily: '"Crimson Text", "Georgia", serif', fontSize: "1rem" }}>
                     Choose your ancestry. Your race determines ability score bonuses, speed, and innate traits.
                  </Typography>

                  <Box sx={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 1.5 }}>
                     {RACES.map((race) => (
                        <Card
                           key={race.id}
                           variant={raceID === race.id ? "outlined" : "elevation"}
                           sx={{ borderColor: raceID === race.id ? "primary.main" : "transparent", borderWidth: 2, cursor: "pointer" }}
                        >
                           <CardActionArea onClick={() => { setRaceID(race.id); setSubraceID(""); }}>
                              <CardContent sx={{ py: 1.5, px: 2, "&:last-child": { pb: 1.5 } }}>
                                 <Typography variant="body2" fontWeight={raceID === race.id ? 700 : 500}>
                                    {race.name}
                                 </Typography>
                                 <Typography variant="caption" color="primary.light" display="block">
                                    {race.asi}
                                 </Typography>
                                 <Typography variant="caption" color="text.secondary" display="block">
                                    Speed: {race.speed}
                                 </Typography>
                                 <Box sx={{ mt: 0.5, display: "flex", flexWrap: "wrap", gap: 0.4 }}>
                                    {race.traits.map((t) => (
                                       <Chip key={t} label={t} size="small" variant="outlined" sx={{ fontSize: "0.6rem", height: 18 }} />
                                    ))}
                                 </Box>
                              </CardContent>
                           </CardActionArea>
                        </Card>
                     ))}
                  </Box>

                  {/* Subrace chips */}
                  {selectedRace && selectedRace.subraces.length > 0 && (
                     <Box>
                        <Typography variant="subtitle2" sx={{ mb: 1, textTransform: "uppercase", letterSpacing: "0.08em" }}>
                           Choose Subrace
                        </Typography>
                        <Box sx={{ display: "flex", flexWrap: "wrap", gap: 1 }}>
                           {selectedRace.subraces.map((sr) => (
                              <Chip
                                 key={sr.id}
                                 label={`${sr.name} (${sr.asiBonusLabel})`}
                                 clickable
                                 onClick={() => setSubraceID(sr.id)}
                                 color={subraceID === sr.id ? "primary" : "default"}
                                 variant={subraceID === sr.id ? "filled" : "outlined"}
                                 sx={{ fontSize: "0.85rem" }}
                              />
                           ))}
                        </Box>
                        {selectedRace.subraces.length > 0 && !subraceID && (
                           <FormHelperText sx={{ color: "warning.main", mt: 0.5 }}>
                              Select a subrace to continue.
                           </FormHelperText>
                        )}
                     </Box>
                  )}

                  {navButtons(!!raceID && (!selectedRace?.subraces.length || !!subraceID), "Next: Class")}
               </Box>
            )}

            {/* ── Step 2: Class ── */}
            {step === STEP.CLASS && (
               <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                  <Typography variant="body2" sx={{ color: "text.secondary", fontStyle: "italic", fontFamily: '"Crimson Text", "Georgia", serif', fontSize: "1rem" }}>
                     Your class defines your combat style, hit points, and special abilities.
                  </Typography>

                  {CLASSES.map((cls) => (
                     <Card
                        key={cls.id}
                        variant={classID === cls.id ? "outlined" : "elevation"}
                        sx={{ borderColor: classID === cls.id ? "primary.main" : "transparent", borderWidth: 2, cursor: "pointer" }}
                     >
                        <CardActionArea onClick={() => { setClassID(cls.id); setSelectedSkills([]); }}>
                           <CardContent>
                              <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 0.5 }}>
                                 <Typography variant="h6" fontWeight={classID === cls.id ? 700 : 500}>
                                    {cls.name}
                                    <Chip label={`Hit Die ${cls.hitDie}`} size="small" sx={{ ml: 1.5, fontSize: "0.7rem" }} />
                                 </Typography>
                                 <Box sx={{ textAlign: "right" }}>
                                    <Typography variant="caption" color="text.secondary" display="block">
                                       Primary: {cls.primaryAbility}
                                    </Typography>
                                    <Typography variant="caption" color="text.secondary" display="block">
                                       Armor: {cls.armorNote}
                                    </Typography>
                                 </Box>
                              </Box>
                              <Typography variant="body2" color="text.secondary" sx={{ fontStyle: "italic", mb: 1 }}>
                                 {cls.flavor}
                              </Typography>
                              <Box sx={{ display: "flex", flexDirection: "column", gap: 0.3 }}>
                                 {cls.features.map((f) => (
                                    <Typography key={f} variant="caption" sx={{ display: "flex", alignItems: "flex-start", gap: 0.5 }}>
                                       <span style={{ color: "gold", flexShrink: 0 }}>•</span> {f}
                                    </Typography>
                                 ))}
                              </Box>
                           </CardContent>
                        </CardActionArea>
                     </Card>
                  ))}

                  {navButtons(!!classID, "Next: Ability Scores")}
               </Box>
            )}

            {/* ── Step 3: Ability Scores ── */}
            {step === STEP.ABILITIES && (
               <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                  <Typography variant="body2" sx={{ color: "text.secondary", fontStyle: "italic", fontFamily: '"Crimson Text", "Georgia", serif', fontSize: "1rem" }}>
                     Assign the standard array (15, 14, 13, 12, 10, 8) to your ability scores. Each value can only be used once.
                     Racial bonuses will be applied automatically.
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

                  {ABILITY_KEYS.map((ab) => {
                     const racial  = racialBonuses[ab] ?? 0;
                     const base    = abilityScores[ab];
                     const total   = base + racial;
                     return (
                        <Box key={ab} sx={{ display: "flex", alignItems: "center", gap: 2 }}>
                           <Typography variant="body2" sx={{ width: 110, fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.05em", flexShrink: 0 }}>
                              {ABILITY_LABELS[ab]}
                           </Typography>
                           <FormControl size="small" sx={{ minWidth: 90 }}>
                              <InputLabel>{ab.toUpperCase()}</InputLabel>
                              <Select
                                 value={base !== 0 ? base : ""}
                                 label={ab.toUpperCase()}
                                 onChange={(e) => assignValue(ab, Number(e.target.value))}
                              >
                                 <MenuItem value=""><em>—</em></MenuItem>
                                 {STANDARD_ARRAY.map((v) => (
                                    <MenuItem key={v} value={v} disabled={usedValues.includes(v) && base !== v}>
                                       {v}
                                    </MenuItem>
                                 ))}
                              </Select>
                           </FormControl>
                           {racial > 0 && (
                              <Typography variant="caption" color="primary.light">
                                 +{racial} racial
                              </Typography>
                           )}
                           {base !== 0 && (
                              <Typography variant="caption" color="text.secondary">
                                 = {total} ({abilityModifier(total)})
                              </Typography>
                           )}
                        </Box>
                     );
                  })}

                  <FormHelperText sx={{ color: usedValues.length === 6 ? "success.main" : "text.secondary" }}>
                     {usedValues.length}/6 values assigned
                  </FormHelperText>

                  {navButtons(usedValues.length === 6, "Next: Skills")}
               </Box>
            )}

            {/* ── Step 4: Skills ── */}
            {step === STEP.SKILLS && selectedClass && (
               <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                  <Typography variant="body2" sx={{ color: "text.secondary", fontStyle: "italic", fontFamily: '"Crimson Text", "Georgia", serif', fontSize: "1rem" }}>
                     Choose {selectedClass.skillCount} skill proficiencies from the {selectedClass.name} list.
                  </Typography>

                  <Box sx={{ display: "flex", flexWrap: "wrap", gap: 1 }}>
                     {selectedClass.skills.map((sk) => {
                        const isSelected = selectedSkills.includes(sk.id);
                        const isDisabled = !isSelected && selectedSkills.length >= selectedClass.skillCount;
                        return (
                           <Chip
                              key={sk.id}
                              label={sk.label}
                              clickable={!isDisabled}
                              onClick={() => !isDisabled && toggleSkill(sk.id)}
                              color={isSelected ? "primary" : "default"}
                              variant={isSelected ? "filled" : "outlined"}
                              sx={{ fontSize: "0.9rem", py: 0.5, opacity: isDisabled ? 0.4 : 1 }}
                           />
                        );
                     })}
                  </Box>

                  <FormHelperText sx={{ color: selectedSkills.length === selectedClass.skillCount ? "success.main" : "text.secondary" }}>
                     {selectedSkills.length}/{selectedClass.skillCount} skills selected
                  </FormHelperText>

                  {navButtons(
                     selectedClass ? selectedSkills.length === selectedClass.skillCount : false,
                     isJoinMode ? "Next: Review" : "Next: Adventure",
                  )}
               </Box>
            )}

            {/* ── Step 5: Adventure Preferences (create mode only) ── */}
            {isAdventureStep && (
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
                     slotProps={{ htmlInput: { maxLength: 200 } }}
                  />

                  {navButtons(true, "Next: Review")}
               </Box>
            )}

            {/* ── Review Step ── */}
            {isReviewStep && selectedClass && (
               <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
                  <Typography variant="body2" sx={{ color: "text.secondary", fontStyle: "italic", fontFamily: '"Crimson Text", "Georgia", serif', fontSize: "1rem" }}>
                     Review your character before entering the world. Once you begin, your race and class are permanent.
                  </Typography>

                  {/* Header row */}
                  <Box sx={{ border: "1px solid rgba(201, 169, 98, 0.3)", borderRadius: 1, p: 2, background: "rgba(106, 78, 157, 0.07)" }}>
                     <Typography variant="h5" sx={{ fontFamily: '"Cinzel", serif', mb: 0.5 }}>{playerName}</Typography>
                     <Typography variant="subtitle1" color="text.secondary">
                        Level 1 {selectedRace?.name}{subraceID && selectedRace ? ` (${selectedRace.subraces.find(s => s.id === subraceID)?.name})` : ""} {selectedClass.name}
                     </Typography>
                     {backstory && (
                        <Typography variant="body2" sx={{ mt: 1, fontStyle: "italic", color: "text.secondary" }}>
                           {backstory}
                        </Typography>
                     )}
                  </Box>

                  {/* Core stats */}
                  <Box sx={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: 1 }}>
                     {[
                        { label: "HP",  value: calcHP() },
                        { label: "AC",  value: calcAC() },
                        { label: "Speed", value: `${selectedRace?.speed ?? "30 ft"}` },
                        { label: "Prof Bonus", value: "+2" },
                     ].map(({ label, value }) => (
                        <Box key={label} sx={{ border: "1px solid rgba(201,169,98,0.25)", borderRadius: 1, p: 1, textAlign: "center" }}>
                           <Typography variant="h6" color="primary.light">{value}</Typography>
                           <Typography variant="caption" color="text.secondary">{label}</Typography>
                        </Box>
                     ))}
                  </Box>

                  {/* Ability scores */}
                  <Box>
                     <Typography variant="subtitle2" sx={{ mb: 1, textTransform: "uppercase", letterSpacing: "0.08em" }}>Ability Scores</Typography>
                     <Box sx={{ display: "grid", gridTemplateColumns: "repeat(6, 1fr)", gap: 1 }}>
                        {ABILITY_KEYS.map((ab) => (
                           <Box key={ab} sx={{ border: "1px solid rgba(201,169,98,0.25)", borderRadius: 1, p: 1, textAlign: "center" }}>
                              <Typography variant="h6">{finalScores[ab]}</Typography>
                              <Typography variant="caption" color="primary.light" display="block">
                                 {abilityModifier(finalScores[ab])}
                              </Typography>
                              <Typography variant="caption" color="text.secondary" sx={{ textTransform: "uppercase", fontSize: "0.6rem" }}>
                                 {ab}
                              </Typography>
                           </Box>
                        ))}
                     </Box>
                  </Box>

                  {/* Skills + saving throws */}
                  <Box sx={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 2 }}>
                     <Box>
                        <Typography variant="subtitle2" sx={{ mb: 0.5, textTransform: "uppercase", letterSpacing: "0.08em" }}>Skill Proficiencies</Typography>
                        {selectedSkills.map((sk) => {
                           const skillDef = selectedClass.skills.find((s) => s.id === sk);
                           return (
                              <Typography key={sk} variant="body2">• {skillDef?.label ?? sk}</Typography>
                           );
                        })}
                     </Box>
                     <Box>
                        <Typography variant="subtitle2" sx={{ mb: 0.5, textTransform: "uppercase", letterSpacing: "0.08em" }}>Saving Throws</Typography>
                        {/* Saving throw proficiencies by class */}
                        {classID === "barbarian" && <><Typography variant="body2">• Strength (+{2 + Math.floor((finalScores.str - 10) / 2)})</Typography><Typography variant="body2">• Constitution (+{2 + Math.floor((finalScores.con - 10) / 2)})</Typography></>}
                        {classID === "fighter"   && <><Typography variant="body2">• Strength (+{2 + Math.floor((finalScores.str - 10) / 2)})</Typography><Typography variant="body2">• Constitution (+{2 + Math.floor((finalScores.con - 10) / 2)})</Typography></>}
                        {classID === "monk"      && <><Typography variant="body2">• Strength (+{2 + Math.floor((finalScores.str - 10) / 2)})</Typography><Typography variant="body2">• Dexterity (+{2 + Math.floor((finalScores.dex - 10) / 2)})</Typography></>}
                     </Box>
                  </Box>

                  {/* Class features */}
                  <Box>
                     <Typography variant="subtitle2" sx={{ mb: 0.5, textTransform: "uppercase", letterSpacing: "0.08em" }}>Class Features</Typography>
                     {selectedClass.features.map((f) => (
                        <Typography key={f} variant="body2">• {f}</Typography>
                     ))}
                  </Box>

                  {/* Adventure preferences (create mode only) */}
                  {!isJoinMode && (themeHint || preferences.length > 0) && (
                     <Box>
                        <Divider sx={{ mb: 1.5 }} />
                        <Typography variant="subtitle2" sx={{ mb: 0.5, textTransform: "uppercase", letterSpacing: "0.08em" }}>Adventure Preferences</Typography>
                        {themeHint && <Typography variant="body2">Theme: {themeHint}</Typography>}
                        {preferences.length > 0 && (
                           <Box sx={{ display: "flex", gap: 0.5, mt: 0.5, flexWrap: "wrap" }}>
                              {preferences.map((p) => <Chip key={p} label={p} size="small" />)}
                           </Box>
                        )}
                     </Box>
                  )}

                  {error && <Alert severity="error">{error}</Alert>}

                  <Box sx={{ display: "flex", justifyContent: "space-between", mt: 1 }}>
                     <Button variant="outlined" onClick={handleBack} disabled={isSubmitting}>Back</Button>
                     <Button variant="contained" onClick={handleSubmit} disabled={isSubmitting} sx={{ minWidth: 180 }}>
                        {isSubmitting
                           ? (isJoinMode ? "Joining..." : "Creating...")
                           : (isJoinMode ? "Enter the Adventure" : "Begin Adventure")}
                     </Button>
                  </Box>
               </Box>
            )}
         </Paper>
      </Box>
   );
}

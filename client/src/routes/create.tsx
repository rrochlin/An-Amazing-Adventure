import {
   createFileRoute,
   redirect,
   useNavigate,
   useSearch,
} from '@tanstack/react-router';
import {
   Alert,
   Box,
   Button,
   Card,
   CardActionArea,
   CardContent,
   Chip,
   CircularProgress,
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
} from '@mui/material';
import { useEffect, useState } from 'react';
import { isAuthenticated } from '@/services/auth.service';
import { CreateGame, JoinCharacter, ListCampaigns } from '@/services/api.game';
import type {
   CampaignManifest,
   CreateGameData,
} from '@/types/types';
import { z } from 'zod';

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
      id: 'human',
      name: 'Human',
      asi: '+1 to all ability scores',
      speed: '30 ft',
      traits: ['Extra Language', 'Versatile'],
      subraces: [],
   },
   {
      id: 'dwarf',
      name: 'Dwarf',
      asi: '+2 CON',
      speed: '25 ft',
      traits: ['Darkvision', 'Dwarven Resilience', 'Stonecunning'],
      subraces: [
         {
            id: 'hill-dwarf',
            name: 'Hill Dwarf',
            asiBonusLabel: '+1 WIS, +1 HP/level',
         },
         {
            id: 'mountain-dwarf',
            name: 'Mountain Dwarf',
            asiBonusLabel: '+2 STR, medium armor',
         },
      ],
   },
   {
      id: 'elf',
      name: 'Elf',
      asi: '+2 DEX',
      speed: '30 ft',
      traits: ['Darkvision', 'Fey Ancestry', 'Trance'],
      subraces: [
         { id: 'high-elf', name: 'High Elf', asiBonusLabel: '+1 INT' },
         {
            id: 'wood-elf',
            name: 'Wood Elf',
            asiBonusLabel: '+1 WIS, 35 ft speed',
         },
      ],
   },
   {
      id: 'halfling',
      name: 'Halfling',
      asi: '+2 DEX',
      speed: '25 ft',
      traits: ['Lucky', 'Brave', 'Halfling Nimbleness'],
      subraces: [
         {
            id: 'lightfoot-halfling',
            name: 'Lightfoot',
            asiBonusLabel: '+1 CHA',
         },
         {
            id: 'stout-halfling',
            name: 'Stout',
            asiBonusLabel: '+1 CON, poison resistance',
         },
      ],
   },
   {
      id: 'dragonborn',
      name: 'Dragonborn',
      asi: '+2 STR, +1 CHA',
      speed: '30 ft',
      traits: ['Draconic Ancestry', 'Breath Weapon', 'Damage Resistance'],
      subraces: [],
   },
   {
      id: 'gnome',
      name: 'Gnome',
      asi: '+2 INT',
      speed: '25 ft',
      traits: ['Darkvision', 'Gnome Cunning'],
      subraces: [
         { id: 'forest-gnome', name: 'Forest Gnome', asiBonusLabel: '+1 DEX' },
         { id: 'rock-gnome', name: 'Rock Gnome', asiBonusLabel: '+1 CON' },
      ],
   },
   // TODO(toolkit): Half-Elf is excluded until rpg-toolkit fixes the racial skill
   // choice ChoiceID bug (SetRace records skills without ChoiceID="half-elf-skills"
   // so ValidateChoices always fails). Re-enable once toolkit is updated.
   // {
   //    id: "half-elf", name: "Half-Elf",
   //    asi: "+2 CHA, +1 to two others", speed: "30 ft",
   //    traits: ["Darkvision", "Fey Ancestry", "Skill Versatility"],
   //    subraces: [],
   // },
   {
      id: 'half-orc',
      name: 'Half-Orc',
      asi: '+2 STR, +1 CON',
      speed: '30 ft',
      traits: [
         'Darkvision',
         'Menacing',
         'Relentless Endurance',
         'Savage Attacks',
      ],
      subraces: [],
   },
   {
      id: 'tiefling',
      name: 'Tiefling',
      asi: '+2 CHA, +1 INT',
      speed: '30 ft',
      traits: ['Darkvision', 'Hellish Resistance', 'Infernal Legacy'],
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
      id: 'barbarian',
      name: 'Barbarian',
      hitDie: 'd12',
      primaryAbility: 'Strength',
      armorNote: 'Light & Medium armor, Shields',
      flavor:
         'A fierce warrior driven by primal rage. The Barbarian trades finesse for raw power — shrugging off wounds that would fell lesser fighters and unleashing devastating attacks while enraged.',
      features: [
         'Rage — +2 bonus damage, resistance to bludgeoning/piercing/slashing (2 rages/day, 1 min)',
         'Unarmored Defense — AC = 10 + DEX modifier + CON modifier (no armor needed)',
         'Reckless Attack — Advantage on STR attacks; enemies also gain advantage against you',
         'Danger Sense — Advantage on DEX saves against visible effects',
         'Extra Attack at level 5 — Attack twice per Attack action',
      ],
      skillCount: 2,
      skills: [
         { id: 'animal-handling', label: 'Animal Handling (WIS)' },
         { id: 'athletics', label: 'Athletics (STR)' },
         { id: 'intimidation', label: 'Intimidation (CHA)' },
         { id: 'nature', label: 'Nature (INT)' },
         { id: 'perception', label: 'Perception (WIS)' },
         { id: 'survival', label: 'Survival (WIS)' },
      ],
   },
   {
      id: 'fighter',
      name: 'Fighter',
      hitDie: 'd10',
      primaryAbility: 'Strength or Dexterity',
      armorNote: 'All armor & Shields',
      flavor:
         'The most versatile warrior on the battlefield. Fighters combine superior weapon mastery with tactical recovery abilities — equally effective with a greatsword or a longbow.',
      features: [
         'Fighting Style — Choose a specialty (Defense, Dueling, Great Weapon, Archery, etc.)',
         'Second Wind — Heal 1d10 + Fighter level HP as a bonus action (1/short rest)',
         'Action Surge — Take one extra full action on your turn (1/short rest)',
         'Extra Attack at level 5 — Attack twice per Attack action',
      ],
      skillCount: 2,
      skills: [
         { id: 'acrobatics', label: 'Acrobatics (DEX)' },
         { id: 'animal-handling', label: 'Animal Handling (WIS)' },
         { id: 'athletics', label: 'Athletics (STR)' },
         { id: 'history', label: 'History (INT)' },
         { id: 'insight', label: 'Insight (WIS)' },
         { id: 'intimidation', label: 'Intimidation (CHA)' },
         { id: 'perception', label: 'Perception (WIS)' },
         { id: 'survival', label: 'Survival (WIS)' },
      ],
   },
   {
      id: 'monk',
      name: 'Monk',
      hitDie: 'd8',
      primaryAbility: 'Dexterity & Wisdom',
      armorNote: 'No armor (AC = 10 + DEX + WIS)',
      flavor:
         'A master of unarmed combat and ki energy. The Monk excels at speed and precision — striking multiple times, deflecting missiles, and moving faster than any armored fighter.',
      features: [
         'Martial Arts — Unarmed strikes deal 1d6 damage and use DEX instead of STR',
         'Ki Points (2/day) — Fuel special abilities; recharge on short rest',
         'Flurry of Blows — 2 bonus unarmed strikes after Attack action (1 Ki)',
         'Patient Defense — Take Dodge as a bonus action (1 Ki)',
         'Step of the Wind — Dash or Disengage as a bonus action, double jump distance (1 Ki)',
         'Extra Attack at level 5 — Attack twice per Attack action',
      ],
      skillCount: 2,
      skills: [
         { id: 'acrobatics', label: 'Acrobatics (DEX)' },
         { id: 'athletics', label: 'Athletics (STR)' },
         { id: 'history', label: 'History (INT)' },
         { id: 'insight', label: 'Insight (WIS)' },
         { id: 'religion', label: 'Religion (INT)' },
         { id: 'stealth', label: 'Stealth (DEX)' },
      ],
   },
];

const STANDARD_ARRAY = [15, 14, 13, 12, 10, 8];
const ABILITY_KEYS = ['str', 'dex', 'con', 'int', 'wis', 'cha'] as const;
type AbilityKey = (typeof ABILITY_KEYS)[number];
const ABILITY_LABELS: Record<AbilityKey, string> = {
   str: 'Strength',
   dex: 'Dexterity',
   con: 'Constitution',
   int: 'Intelligence',
   wis: 'Wisdom',
   cha: 'Charisma',
};

// Racial ASI bonuses keyed by race_id and optional subrace_id.
// Keys must match the toolkit's kebab-case constants exactly.
const RACIAL_ASI: Record<string, Partial<Record<AbilityKey, number>>> = {
   human: { str: 1, dex: 1, con: 1, int: 1, wis: 1, cha: 1 },
   dwarf: { con: 2 },
   'hill-dwarf': { wis: 1 },
   'mountain-dwarf': { str: 2 },
   elf: { dex: 2 },
   'high-elf': { int: 1 },
   'wood-elf': { wis: 1 },
   halfling: { dex: 2 },
   'lightfoot-halfling': { cha: 1 },
   'stout-halfling': { con: 1 },
   dragonborn: { str: 2, cha: 1 },
   gnome: { int: 2 },
   'forest-gnome': { dex: 1 },
   'rock-gnome': { con: 1 },
   // "half-elf" excluded — see TODO above
   'half-orc': { str: 2, con: 1 },
   tiefling: { cha: 2, int: 1 },
};

function getRacialBonuses(
   raceId: string,
   subraceId: string,
): Partial<Record<AbilityKey, number>> {
   const base = RACIAL_ASI[raceId] ?? {};
   const sub = subraceId ? (RACIAL_ASI[subraceId] ?? {}) : {};
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

// ─── Steps ──────────────────────────────────────────────────────────────────

const CREATE_STEPS = [
   'Campaign',
   'Name',
   'Race',
   'Class',
   'Ability Scores',
   'Skills',
   'Review',
];
const JOIN_STEPS = [
   'Name',
   'Race',
   'Class',
   'Ability Scores',
   'Skills',
   'Review',
];

const CREATE_STEP = {
   CAMPAIGN: 0,
   NAME: 1,
   RACE: 2,
   CLASS: 3,
   ABILITIES: 4,
   SKILLS: 5,
   REVIEW: 6,
} as const;

const JOIN_STEP = {
   CAMPAIGN: -1,
   NAME: 0,
   RACE: 1,
   CLASS: 2,
   ABILITIES: 3,
   SKILLS: 4,
   REVIEW: 5,
} as const;

// ─── Route ──────────────────────────────────────────────────────────────────

export const Route = createFileRoute('/create')({
   component: CreateRoute,
   validateSearch: z.object({ session: z.string().optional() }),
   beforeLoad: () => {
      if (!isAuthenticated()) {
         throw redirect({ to: '/login', search: { redirect: location.href } });
      }
   },
});

// ─── Component ──────────────────────────────────────────────────────────────

/** Exported for unit testing (bypasses TanStack route lazy wrapper). */
export function CreateRoute() {
   const navigate = useNavigate();
   const { session: joinSessionId } = useSearch({ from: '/create' });
   const isJoinMode = !!joinSessionId;
   const STEPS = isJoinMode ? JOIN_STEPS : CREATE_STEPS;
   const STEP = isJoinMode ? JOIN_STEP : CREATE_STEP;

   const [step, setStep] = useState(0);
   const [error, setError] = useState<string | null>(null);
   const [isSubmitting, setIsSubmitting] = useState(false);

   // Step 0/1 — Name + Backstory
   const [playerName, setPlayerName] = useState('');
   const [backstory, setBackstory] = useState('');

   // Step 2 — Race
   const [raceID, setRaceID] = useState('');
   const [subraceID, setSubraceID] = useState('');

   // Step 3 — Class
   const [classID, setClassID] = useState('');

   // Step 4 — Ability Scores (standard array)
   const [abilityScores, setAbilityScores] = useState<
      Record<AbilityKey, number>
   >({
      str: 0,
      dex: 0,
      con: 0,
      int: 0,
      wis: 0,
      cha: 0,
   });
   // Derived — no separate state; avoids stale-closure bugs on swap.
   const usedValues = Object.values(abilityScores).filter((v) => v !== 0);

   // Step 5 — Skills
   const [selectedSkills, setSelectedSkills] = useState<string[]>([]);

   // Step 0 — Campaign selection (create mode only)
   const [availableCampaigns, setAvailableCampaigns] = useState<
      CampaignManifest[]
   >([]);
   const [selectedCampaignID, setSelectedCampaignID] = useState('');
   const [isLoadingCampaigns, setIsLoadingCampaigns] = useState(false);
   const [campaignLoadError, setCampaignLoadError] = useState<string | null>(
      null,
   );

   // ── Derived ──
   const selectedRace = RACES.find((r) => r.id === raceID);
   const selectedClass = CLASSES.find((c) => c.id === classID);
   const selectedCampaign = availableCampaigns.find(
      (campaign) => campaign.id === selectedCampaignID,
   );
   const racialBonuses = getRacialBonuses(raceID, subraceID);

   useEffect(() => {
      if (isJoinMode) return;

      let cancelled = false;
      setIsLoadingCampaigns(true);
      setCampaignLoadError(null);

      ListCampaigns()
         .then((response) => {
            if (cancelled) return;
            if (!Array.isArray(response.campaigns)) {
               throw new Error(
                  'Failed to load campaigns. The server returned an invalid response.',
               );
            }
            setAvailableCampaigns(response.campaigns);
         })
         .catch((e: unknown) => {
            if (cancelled) return;
            const msg =
               e instanceof Error
                  ? e.message
                  : 'Failed to load campaigns. Please try again later.';
            setCampaignLoadError(msg);
         })
         .finally(() => {
            if (cancelled) return;
            setIsLoadingCampaigns(false);
         });

      return () => {
         cancelled = true;
      };
   }, [isJoinMode]);

   // Final scores (base + racial bonus) used in Review step
   const finalScores = ABILITY_KEYS.reduce<Record<AbilityKey, number>>(
      (acc, ab) => {
         acc[ab] = (abilityScores[ab] || 0) + (racialBonuses[ab] ?? 0);
         return acc;
      },
      { str: 0, dex: 0, con: 0, int: 0, wis: 0, cha: 0 },
   );

   // HP at level 1: hit die max + CON mod
   function calcHP(): number {
      if (!selectedClass) return 0;
      const hitDieMax = { d12: 12, d10: 10, d8: 8 }[selectedClass.hitDie] ?? 8;
      const conMod = Math.floor((finalScores.con - 10) / 2);
      return hitDieMax + conMod;
   }

   // AC at level 1: Monk uses 10 + DEX + WIS (unarmored defense), others use 10 + DEX
   function calcAC(): number {
      const dexMod = Math.floor((finalScores.dex - 10) / 2);
      if (classID === 'monk') {
         const wisMod = Math.floor((finalScores.wis - 10) / 2);
         return 10 + dexMod + wisMod;
      }
      return 10 + dexMod;
   }

   // ── Ability score helpers ──
   const assignValue = (ability: AbilityKey, value: number) => {
      setAbilityScores((prev) => {
         const next = { ...prev };
         // If the chosen value is already assigned to another ability, swap: clear that slot.
         if (value !== 0) {
            for (const key of ABILITY_KEYS) {
               if (key !== ability && next[key] === value) {
                  next[key] = 0;
               }
            }
         }
         next[ability] = value;
         return next;
      });
   };

   const randomizeAbilityScores = () => {
      const shuffled = [...STANDARD_ARRAY].sort(() => Math.random() - 0.5);
      const next = {} as Record<AbilityKey, number>;
      ABILITY_KEYS.forEach((key, i) => {
         next[key] = shuffled[i];
      });
      setAbilityScores(next);
   };

   // ── Skill helpers ──
   const toggleSkill = (skillId: string) => {
      setSelectedSkills((prev) => {
         if (prev.includes(skillId)) return prev.filter((s) => s !== skillId);
         if (selectedClass && prev.length >= selectedClass.skillCount)
            return prev;
         return [...prev, skillId];
      });
   };

   // ── Test campaign quick-start ──
   // Randomizes all required fields using only the test campaign's allowed
   // races/classes and submits directly — skips the entire wizard.
   const TEST_ALLOWED_RACES = ['human', 'dwarf'] as const;
   const TEST_DWARF_SUBRACES = ['hill-dwarf', 'mountain-dwarf'];
   const TEST_ALLOWED_CLASS_IDS = ['fighter', 'barbarian', 'monk'];

   const handleTestQuickStart = async () => {
      if (isSubmitting) return;
      setError(null);
      setIsSubmitting(true);
      try {
         const suffix = Math.random().toString(36).slice(2, 6).toUpperCase();
         const name = `Tester-${suffix}`;

         const pickedRace =
            TEST_ALLOWED_RACES[
               Math.floor(Math.random() * TEST_ALLOWED_RACES.length)
            ];
         const pickedSubrace =
            pickedRace === 'dwarf'
               ? TEST_DWARF_SUBRACES[
                    Math.floor(Math.random() * TEST_DWARF_SUBRACES.length)
                 ]
               : undefined;

         const pickedClassID =
            TEST_ALLOWED_CLASS_IDS[
               Math.floor(Math.random() * TEST_ALLOWED_CLASS_IDS.length)
            ];
         const pickedClass = CLASSES.find((c) => c.id === pickedClassID)!;

         const shuffledScores = [...STANDARD_ARRAY].sort(
            () => Math.random() - 0.5,
         );
         const scores = {} as Record<AbilityKey, number>;
         ABILITY_KEYS.forEach((key, i) => {
            scores[key] = shuffledScores[i];
         });

         const shuffledSkills = [...pickedClass.skills].sort(
            () => Math.random() - 0.5,
         );
         const skills = shuffledSkills
            .slice(0, pickedClass.skillCount)
            .map((s) => s.id);

         const result = await CreateGame({
            name,
            race_id: pickedRace,
            subrace_id: pickedSubrace,
            class_id: pickedClassID,
            ability_scores: scores,
            selected_skills: skills,
            campaign_id: 'test',
         });
         navigate({
            to: '/game-{$sessionUUID}',
            params: { sessionUUID: result.session_id },
         });
      } catch (e: unknown) {
         const msg =
            e instanceof Error
               ? e.message
               : 'Quick start failed — please try again.';
         setError(msg);
         setIsSubmitting(false);
      }
   };

   const handleNext = () => {
      setStep((s) => s + 1);
      setError(null);
   };
   const handleBack = () => {
      setStep((s) => s - 1);
      setError(null);
   };

   // ── Determine if current step is the last action step before submit ──
   const isReviewStep = step === STEP.REVIEW;
   const isCampaignStep = step === STEP.CAMPAIGN;

   // ── Submit ──
   const buildPayload = (): CreateGameData => ({
      name: playerName.trim(),
      backstory: backstory.trim() || undefined,
      race_id: raceID,
      subrace_id: subraceID || undefined,
      class_id: classID,
      ability_scores: abilityScores,
      selected_skills: selectedSkills,
      campaign_id: selectedCampaignID,
   });

   const handleSubmit = async () => {
      setError(null);
      setIsSubmitting(true);
      try {
         const payload = buildPayload();
         if (isJoinMode) {
            await JoinCharacter(joinSessionId, payload);
            navigate({
               to: '/game-{$sessionUUID}',
               params: { sessionUUID: joinSessionId },
            });
            return;
         }
         const result = await CreateGame(payload);
         navigate({
            to: '/game-{$sessionUUID}',
            params: { sessionUUID: result.session_id },
         });
      } catch (e: unknown) {
         const msg =
            e instanceof Error
               ? e.message
               : 'Failed to create adventure — please try again.';
         setError(msg);
         setIsSubmitting(false);
      }
   };

   // ── Nav helpers ──
   const navButtons = (canNext: boolean, nextLabel?: string) => (
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mt: 3 }}>
         <Button
            variant="outlined"
            onClick={step === 0 ? () => navigate({ to: '/' }) : handleBack}
         >
            {step === 0 ? 'Cancel' : 'Back'}
         </Button>
         <Button variant="contained" onClick={handleNext} disabled={!canNext}>
            {nextLabel ?? 'Next'}
         </Button>
      </Box>
   );

   // ── Render ──
   return (
      <Box
         sx={{
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'flex-start',
            minHeight: 'calc(100vh - 64px)',
            p: { xs: 2, sm: 4 },
            pt: 6,
         }}
      >
         <Paper
            sx={{
               maxWidth: 800,
               width: '100%',
               p: { xs: 2, sm: 4 },
               backgroundImage:
                  'linear-gradient(rgba(106, 78, 157, 0.05), rgba(201, 169, 98, 0.05))',
               border: '1px solid rgba(201, 169, 98, 0.2)',
            }}
         >
            <Typography
               variant="h3"
               sx={{
                  mb: 1,
                  textAlign: 'center',
                  textTransform: 'uppercase',
                  letterSpacing: '0.1em',
                  fontSize: '2rem',
                  borderBottom: '3px solid',
                  borderColor: 'primary.main',
                  pb: 2,
               }}
            >
               {isJoinMode ? 'Join the Adventure' : 'Forge Your Adventure'}
            </Typography>
            {isJoinMode && (
               <Alert severity="info" sx={{ mb: 2 }}>
                  You&apos;re joining an existing adventure. Create your
                  character and you&apos;ll be dropped into the world.
               </Alert>
            )}

            <Stepper activeStep={step} sx={{ mb: 4, mt: 3 }} alternativeLabel>
               {STEPS.map((label) => (
                  <Step key={label}>
                     <StepLabel>{label}</StepLabel>
                  </Step>
               ))}
            </Stepper>

            {/* ── Step 0: Campaign Selection (create mode only) ── */}
            {isCampaignStep && (
               <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
                  <Typography
                     variant="body2"
                     sx={{
                        color: 'text.secondary',
                        fontStyle: 'italic',
                        fontFamily: '"Crimson Text", "Georgia", serif',
                        fontSize: '1rem',
                     }}
                  >
                     Choose which authored campaign to enter before building your
                     character. This lets the campaign shape the rest of the
                     create flow.
                  </Typography>

                  <Box>
                     <Typography
                        variant="subtitle2"
                        sx={{
                           mb: 1.5,
                           textTransform: 'uppercase',
                           letterSpacing: '0.08em',
                        }}
                     >
                        Campaign
                     </Typography>
                     <FormControl fullWidth>
                        <InputLabel id="campaign-select-label">
                           Campaign
                        </InputLabel>
                        <Select
                           labelId="campaign-select-label"
                           label="Campaign"
                           value={selectedCampaignID}
                           onChange={(e) => setSelectedCampaignID(e.target.value)}
                           disabled={isLoadingCampaigns}
                        >
                           <MenuItem value="" disabled>
                              Select a campaign
                           </MenuItem>
                           {availableCampaigns.map((campaign) => (
                              <MenuItem key={campaign.id} value={campaign.id}>
                                 {campaign.title}
                              </MenuItem>
                           ))}
                        </Select>
                        <FormHelperText>
                           {isLoadingCampaigns
                              ? 'Loading available campaigns...'
                              : availableCampaigns.length === 0
                                ? 'No campaigns are currently available.'
                                : 'Pick one of the available premade campaigns.'}
                        </FormHelperText>
                     </FormControl>
                  </Box>

                  {campaignLoadError && (
                     <Alert severity="error">{campaignLoadError}</Alert>
                  )}

                  {selectedCampaign && (
                     <Paper
                        variant="outlined"
                        sx={{
                           p: 2,
                           background: 'rgba(106, 78, 157, 0.07)',
                           borderColor: 'rgba(201, 169, 98, 0.3)',
                        }}
                     >
                        <Typography
                           variant="h6"
                           sx={{ fontFamily: '"Cinzel", serif', mb: 0.5 }}
                        >
                           {selectedCampaign.title}
                        </Typography>
                        <Typography variant="body2" sx={{ mb: 1 }}>
                           {selectedCampaign.premise}
                        </Typography>
                        {selectedCampaign.description && (
                           <Typography
                              variant="body2"
                              color="text.secondary"
                              sx={{ mb: selectedCampaign.tone ? 1 : 0 }}
                           >
                              {selectedCampaign.description}
                           </Typography>
                        )}
                        {selectedCampaign.tone && (
                           <Chip
                              label={`Tone: ${selectedCampaign.tone}`}
                              size="small"
                              variant="outlined"
                           />
                        )}
                     </Paper>
                  )}

                  {/* Quick Start shortcut — only shown for the test campaign */}
                  {selectedCampaignID === 'test' && (
                     <Box>
                        <Divider sx={{ mb: 2 }}>
                           <Typography
                              variant="caption"
                              sx={{
                                 textTransform: 'uppercase',
                                 letterSpacing: '0.08em',
                                 color: 'text.secondary',
                              }}
                           >
                              or
                           </Typography>
                        </Divider>
                        <Button
                           variant="contained"
                           color="secondary"
                           fullWidth
                           onClick={handleTestQuickStart}
                           disabled={isSubmitting}
                           sx={{ fontFamily: '"Cinzel", serif' }}
                        >
                           {isSubmitting ? (
                              <CircularProgress size={20} sx={{ color: 'inherit' }} />
                           ) : (
                              '⚡ Quick Start — Randomize & Play'
                           )}
                        </Button>
                        <Typography
                           variant="caption"
                           color="text.secondary"
                           sx={{ display: 'block', textAlign: 'center', mt: 0.75 }}
                        >
                           Picks a random name, race, class, ability scores, and skills — skips the wizard
                        </Typography>
                     </Box>
                  )}

                  {navButtons(!!selectedCampaignID, 'Next: Character')}
               </Box>
            )}

            {/* ── Step 1: Name + Backstory ── */}
            {step === STEP.NAME && (
               <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
                  <TextField
                     label="Character Name"
                     value={playerName}
                     onChange={(e) => setPlayerName(e.target.value)}
                     fullWidth
                     required
                     helperText="What are you called, adventurer? (2–40 characters)"
                     slotProps={{
                        htmlInput: { maxLength: 40, autoComplete: 'off' },
                     }}
                  />
                  <TextField
                     label="Backstory (optional)"
                     value={backstory}
                     onChange={(e) => setBackstory(e.target.value)}
                     fullWidth
                     multiline
                     rows={3}
                     helperText="A brief 2–3 sentence origin for your character. The Dungeon Master will weave it into the story."
                     slotProps={{ htmlInput: { maxLength: 300 } }}
                  />
                  {navButtons(playerName.trim().length >= 2, 'Next: Race')}
               </Box>
            )}

            {/* ── Step 2: Race ── */}
            {step === STEP.RACE && (
               <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
                  <Typography
                     variant="body2"
                     sx={{
                        color: 'text.secondary',
                        fontStyle: 'italic',
                        fontFamily: '"Crimson Text", "Georgia", serif',
                        fontSize: '1rem',
                     }}
                  >
                     Choose your ancestry. Your race determines ability score
                     bonuses, speed, and innate traits.
                  </Typography>

                  <Box
                     sx={{
                        display: 'grid',
                        gridTemplateColumns: 'repeat(3, 1fr)',
                        gap: 1.5,
                     }}
                  >
                     {RACES.map((race) => {
                        const isSelected = raceID === race.id;
                        const hasSubraces = race.subraces.length > 0;
                        return (
                           <Card
                              key={race.id}
                              variant={isSelected ? 'outlined' : 'elevation'}
                              sx={{
                                 borderColor: isSelected
                                    ? 'primary.main'
                                    : 'transparent',
                                 borderWidth: 2,
                                 cursor: 'pointer',
                              }}
                           >
                              <CardActionArea
                                 onClick={() => {
                                    setRaceID(race.id);
                                    setSubraceID('');
                                 }}
                              >
                                 <CardContent
                                    sx={{
                                       py: 1.5,
                                       px: 2,
                                       '&:last-child': {
                                          pb:
                                             isSelected && hasSubraces
                                                ? 1
                                                : 1.5,
                                       },
                                    }}
                                 >
                                    <Typography
                                       variant="body2"
                                       fontWeight={isSelected ? 700 : 500}
                                    >
                                       {race.name}
                                    </Typography>
                                    <Typography
                                       variant="caption"
                                       color="primary.light"
                                       display="block"
                                    >
                                       {race.asi}
                                    </Typography>
                                    <Typography
                                       variant="caption"
                                       color="text.secondary"
                                       display="block"
                                    >
                                       Speed: {race.speed}
                                    </Typography>
                                    <Box
                                       sx={{
                                          mt: 0.5,
                                          display: 'flex',
                                          flexWrap: 'wrap',
                                          gap: 0.4,
                                       }}
                                    >
                                       {race.traits.map((t) => (
                                          <Chip
                                             key={t}
                                             label={t}
                                             size="small"
                                             variant="outlined"
                                             sx={{
                                                fontSize: '0.6rem',
                                                height: 18,
                                             }}
                                          />
                                       ))}
                                       {!isSelected && hasSubraces && (
                                          <Chip
                                             label="has subraces"
                                             size="small"
                                             color="secondary"
                                             variant="outlined"
                                             sx={{
                                                fontSize: '0.6rem',
                                                height: 18,
                                             }}
                                          />
                                       )}
                                    </Box>
                                 </CardContent>
                              </CardActionArea>
                              {/* Subrace picker — shown inline on the selected card */}
                              {isSelected && hasSubraces && (
                                 <CardContent
                                    sx={{
                                       pt: 0,
                                       pb: 1.5,
                                       px: 2,
                                       '&:last-child': { pb: 1.5 },
                                    }}
                                 >
                                    <Divider sx={{ mb: 1 }} />
                                    <Typography
                                       variant="caption"
                                       sx={{
                                          textTransform: 'uppercase',
                                          letterSpacing: '0.08em',
                                          color: 'text.secondary',
                                          display: 'block',
                                          mb: 0.75,
                                       }}
                                    >
                                       Choose Subrace
                                    </Typography>
                                    <Box
                                       sx={{
                                          display: 'flex',
                                          flexWrap: 'wrap',
                                          gap: 0.75,
                                       }}
                                    >
                                       {race.subraces.map((sr) => (
                                          <Chip
                                             key={sr.id}
                                             label={`${sr.name} (${sr.asiBonusLabel})`}
                                             clickable
                                             onClick={(e) => {
                                                e.stopPropagation();
                                                setSubraceID(sr.id);
                                             }}
                                             color={
                                                subraceID === sr.id
                                                   ? 'primary'
                                                   : 'default'
                                             }
                                             variant={
                                                subraceID === sr.id
                                                   ? 'filled'
                                                   : 'outlined'
                                             }
                                             size="small"
                                             sx={{ fontSize: '0.8rem' }}
                                          />
                                       ))}
                                    </Box>
                                    {!subraceID && (
                                       <FormHelperText
                                          sx={{
                                             color: 'warning.main',
                                             mt: 0.5,
                                          }}
                                       >
                                          Select a subrace to continue.
                                       </FormHelperText>
                                    )}
                                 </CardContent>
                              )}
                           </Card>
                        );
                     })}
                  </Box>

                  {navButtons(
                     !!raceID &&
                        (!selectedRace?.subraces.length || !!subraceID),
                     'Next: Class',
                  )}
               </Box>
            )}

            {/* ── Step 3: Class ── */}
            {step === STEP.CLASS && (
               <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                  <Typography
                     variant="body2"
                     sx={{
                        color: 'text.secondary',
                        fontStyle: 'italic',
                        fontFamily: '"Crimson Text", "Georgia", serif',
                        fontSize: '1rem',
                     }}
                  >
                     Your class defines your combat style, hit points, and
                     special abilities.
                  </Typography>

                  {CLASSES.map((cls) => (
                     <Card
                        key={cls.id}
                        variant={classID === cls.id ? 'outlined' : 'elevation'}
                        sx={{
                           borderColor:
                              classID === cls.id
                                 ? 'primary.main'
                                 : 'transparent',
                           borderWidth: 2,
                           cursor: 'pointer',
                        }}
                     >
                        <CardActionArea
                           onClick={() => {
                              setClassID(cls.id);
                              setSelectedSkills([]);
                           }}
                        >
                           <CardContent>
                              <Box
                                 sx={{
                                    display: 'flex',
                                    justifyContent: 'space-between',
                                    alignItems: 'center',
                                    mb: 0.5,
                                 }}
                              >
                                 <Typography
                                    variant="h6"
                                    fontWeight={classID === cls.id ? 700 : 500}
                                 >
                                    {cls.name}
                                    <Chip
                                       label={`Hit Die ${cls.hitDie}`}
                                       size="small"
                                       sx={{ ml: 1.5, fontSize: '0.7rem' }}
                                    />
                                 </Typography>
                                 <Box sx={{ textAlign: 'right' }}>
                                    <Typography
                                       variant="caption"
                                       color="text.secondary"
                                       display="block"
                                    >
                                       Primary: {cls.primaryAbility}
                                    </Typography>
                                    <Typography
                                       variant="caption"
                                       color="text.secondary"
                                       display="block"
                                    >
                                       Armor: {cls.armorNote}
                                    </Typography>
                                 </Box>
                              </Box>
                              <Typography
                                 variant="body2"
                                 color="text.secondary"
                                 sx={{ fontStyle: 'italic', mb: 1 }}
                              >
                                 {cls.flavor}
                              </Typography>
                              <Box
                                 sx={{
                                    display: 'flex',
                                    flexDirection: 'column',
                                    gap: 0.3,
                                 }}
                              >
                                 {cls.features.map((f) => (
                                    <Typography
                                       key={f}
                                       variant="caption"
                                       sx={{
                                          display: 'flex',
                                          alignItems: 'flex-start',
                                          gap: 0.5,
                                       }}
                                    >
                                       <span
                                          style={{
                                             color: 'gold',
                                             flexShrink: 0,
                                          }}
                                       >
                                          •
                                       </span>{' '}
                                       {f}
                                    </Typography>
                                 ))}
                              </Box>
                           </CardContent>
                        </CardActionArea>
                     </Card>
                  ))}

                  {navButtons(!!classID, 'Next: Ability Scores')}
               </Box>
            )}

            {/* ── Step 4: Ability Scores ── */}
            {step === STEP.ABILITIES && (
               <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                  <Typography
                     variant="body2"
                     sx={{
                        color: 'text.secondary',
                        fontStyle: 'italic',
                        fontFamily: '"Crimson Text", "Georgia", serif',
                        fontSize: '1rem',
                     }}
                  >
                     Assign the standard array (15, 14, 13, 12, 10, 8) to your
                     ability scores. Each value can only be used once. Racial
                     bonuses will be applied automatically.
                  </Typography>

                  <Box
                     sx={{
                        display: 'flex',
                        flexWrap: 'wrap',
                        gap: 1,
                        mb: 1,
                        alignItems: 'center',
                     }}
                  >
                     <Typography
                        variant="caption"
                        sx={{ width: '100%', color: 'text.secondary' }}
                     >
                        Available values:
                     </Typography>
                     {STANDARD_ARRAY.map((v) => (
                        <Chip
                           key={v}
                           label={v}
                           variant={
                              usedValues.includes(v) ? 'filled' : 'outlined'
                           }
                           color={
                              usedValues.includes(v) ? 'default' : 'primary'
                           }
                           size="small"
                        />
                     ))}
                     <Button
                        size="small"
                        variant="outlined"
                        onClick={randomizeAbilityScores}
                        sx={{ ml: 'auto', fontSize: '0.75rem' }}
                     >
                        Randomize
                     </Button>
                  </Box>

                  {ABILITY_KEYS.map((ab) => {
                     const racial = racialBonuses[ab] ?? 0;
                     const base = abilityScores[ab];
                     const total = base + racial;
                     return (
                        <Box
                           key={ab}
                           sx={{
                              display: 'flex',
                              alignItems: 'center',
                              gap: 1.5,
                              flexWrap: 'nowrap',
                              minWidth: 0,
                           }}
                        >
                           <Typography
                              variant="body2"
                              sx={{
                                 width: 100,
                                 minWidth: 100,
                                 fontWeight: 600,
                                 textTransform: 'uppercase',
                                 letterSpacing: '0.05em',
                                 flexShrink: 0,
                                 fontSize: '0.85rem',
                              }}
                           >
                              {ABILITY_LABELS[ab]}
                           </Typography>
                           <FormControl
                              size="small"
                              sx={{ width: 80, flexShrink: 0 }}
                           >
                              <InputLabel shrink>{ab.toUpperCase()}</InputLabel>
                              <Select
                                 value={base !== 0 ? base : ''}
                                 label={ab.toUpperCase()}
                                 notched
                                 displayEmpty
                                 onChange={(e) =>
                                    assignValue(ab, Number(e.target.value))
                                 }
                              >
                                 <MenuItem value="">
                                    <em>—</em>
                                 </MenuItem>
                                 {STANDARD_ARRAY.map((v) => (
                                    <MenuItem key={v} value={v}>
                                       {v}
                                       {usedValues.includes(v) && base !== v
                                          ? ' ↔'
                                          : ''}
                                    </MenuItem>
                                 ))}
                              </Select>
                           </FormControl>
                           <Box
                              sx={{
                                 display: 'flex',
                                 alignItems: 'center',
                                 gap: 0.75,
                                 flexShrink: 0,
                              }}
                           >
                              {racial > 0 && (
                                 <Typography
                                    variant="caption"
                                    color="primary.light"
                                    sx={{ whiteSpace: 'nowrap' }}
                                 >
                                    +{racial}
                                 </Typography>
                              )}
                              {base !== 0 && (
                                 <Typography
                                    variant="caption"
                                    color="text.secondary"
                                    sx={{ whiteSpace: 'nowrap' }}
                                 >
                                    = {total} ({abilityModifier(total)})
                                 </Typography>
                              )}
                           </Box>
                        </Box>
                     );
                  })}

                  <FormHelperText
                     sx={{
                        color:
                           usedValues.length === 6
                              ? 'success.main'
                              : 'text.secondary',
                     }}
                  >
                     {usedValues.length}/6 values assigned
                  </FormHelperText>

                  {navButtons(usedValues.length === 6, 'Next: Skills')}
               </Box>
            )}

            {/* ── Step 5: Skills ── */}
            {step === STEP.SKILLS && selectedClass && (
               <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                  <Typography
                     variant="body2"
                     sx={{
                        color: 'text.secondary',
                        fontStyle: 'italic',
                        fontFamily: '"Crimson Text", "Georgia", serif',
                        fontSize: '1rem',
                     }}
                  >
                     Choose {selectedClass.skillCount} skill proficiencies from
                     the {selectedClass.name} list.
                  </Typography>

                  <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                     {selectedClass.skills.map((sk) => {
                        const isSelected = selectedSkills.includes(sk.id);
                        const isDisabled =
                           !isSelected &&
                           selectedSkills.length >= selectedClass.skillCount;
                        return (
                           <Chip
                              key={sk.id}
                              label={sk.label}
                              clickable={!isDisabled}
                              onClick={() => !isDisabled && toggleSkill(sk.id)}
                              color={isSelected ? 'primary' : 'default'}
                              variant={isSelected ? 'filled' : 'outlined'}
                              sx={{
                                 fontSize: '0.9rem',
                                 py: 0.5,
                                 opacity: isDisabled ? 0.4 : 1,
                              }}
                           />
                        );
                     })}
                  </Box>

                  <FormHelperText
                     sx={{
                        color:
                           selectedSkills.length === selectedClass.skillCount
                              ? 'success.main'
                              : 'text.secondary',
                     }}
                  >
                     {selectedSkills.length}/{selectedClass.skillCount} skills
                     selected
                  </FormHelperText>

                  {navButtons(
                     selectedClass
                         ? selectedSkills.length === selectedClass.skillCount
                          : false,
                     'Next: Review',
                   )}
                </Box>
             )}

            {/* ── Review Step ── */}
            {isReviewStep && selectedClass && (
               <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
                  <Typography
                     variant="body2"
                     sx={{
                        color: 'text.secondary',
                        fontStyle: 'italic',
                        fontFamily: '"Crimson Text", "Georgia", serif',
                        fontSize: '1rem',
                     }}
                  >
                     Review your character before entering the world. Once you
                     begin, your race and class are permanent.
                  </Typography>

                  {/* Header row */}
                  <Box
                     sx={{
                        border: '1px solid rgba(201, 169, 98, 0.3)',
                        borderRadius: 1,
                        p: 2,
                        background: 'rgba(106, 78, 157, 0.07)',
                     }}
                  >
                     <Typography
                        variant="h5"
                        sx={{ fontFamily: '"Cinzel", serif', mb: 0.5 }}
                     >
                        {playerName}
                     </Typography>
                     <Typography variant="subtitle1" color="text.secondary">
                        Level 1 {selectedRace?.name}
                        {subraceID && selectedRace
                           ? ` (${selectedRace.subraces.find((s) => s.id === subraceID)?.name})`
                           : ''}{' '}
                        {selectedClass.name}
                     </Typography>
                     {backstory && (
                        <Typography
                           variant="body2"
                           sx={{
                              mt: 1,
                              fontStyle: 'italic',
                              color: 'text.secondary',
                           }}
                        >
                           {backstory}
                        </Typography>
                     )}
                  </Box>

                  {/* Core stats */}
                  <Box
                     sx={{
                        display: 'grid',
                        gridTemplateColumns: 'repeat(4, 1fr)',
                        gap: 1,
                     }}
                  >
                     {[
                        { label: 'HP', value: calcHP() },
                        { label: 'AC', value: calcAC() },
                        {
                           label: 'Speed',
                           value: `${selectedRace?.speed ?? '30 ft'}`,
                        },
                        { label: 'Prof Bonus', value: '+2' },
                     ].map(({ label, value }) => (
                        <Box
                           key={label}
                           sx={{
                              border: '1px solid rgba(201,169,98,0.25)',
                              borderRadius: 1,
                              p: 1,
                              textAlign: 'center',
                           }}
                        >
                           <Typography variant="h6" color="primary.light">
                              {value}
                           </Typography>
                           <Typography variant="caption" color="text.secondary">
                              {label}
                           </Typography>
                        </Box>
                     ))}
                  </Box>

                  {/* Ability scores */}
                  <Box>
                     <Typography
                        variant="subtitle2"
                        sx={{
                           mb: 1,
                           textTransform: 'uppercase',
                           letterSpacing: '0.08em',
                        }}
                     >
                        Ability Scores
                     </Typography>
                     <Box
                        sx={{
                           display: 'grid',
                           gridTemplateColumns: 'repeat(6, 1fr)',
                           gap: 1,
                        }}
                     >
                        {ABILITY_KEYS.map((ab) => (
                           <Box
                              key={ab}
                              sx={{
                                 border: '1px solid rgba(201,169,98,0.25)',
                                 borderRadius: 1,
                                 p: 1,
                                 textAlign: 'center',
                              }}
                           >
                              <Typography variant="h6">
                                 {finalScores[ab]}
                              </Typography>
                              <Typography
                                 variant="caption"
                                 color="primary.light"
                                 display="block"
                              >
                                 {abilityModifier(finalScores[ab])}
                              </Typography>
                              <Typography
                                 variant="caption"
                                 color="text.secondary"
                                 sx={{
                                    textTransform: 'uppercase',
                                    fontSize: '0.6rem',
                                 }}
                              >
                                 {ab}
                              </Typography>
                           </Box>
                        ))}
                     </Box>
                  </Box>

                  {/* Skills + saving throws */}
                  <Box
                     sx={{
                        display: 'grid',
                        gridTemplateColumns: '1fr 1fr',
                        gap: 2,
                     }}
                  >
                     <Box>
                        <Typography
                           variant="subtitle2"
                           sx={{
                              mb: 0.5,
                              textTransform: 'uppercase',
                              letterSpacing: '0.08em',
                           }}
                        >
                           Skill Proficiencies
                        </Typography>
                        {selectedSkills.map((sk) => {
                           const skillDef = selectedClass.skills.find(
                              (s) => s.id === sk,
                           );
                           return (
                              <Typography key={sk} variant="body2">
                                 • {skillDef?.label ?? sk}
                              </Typography>
                           );
                        })}
                     </Box>
                     <Box>
                        <Typography
                           variant="subtitle2"
                           sx={{
                              mb: 0.5,
                              textTransform: 'uppercase',
                              letterSpacing: '0.08em',
                           }}
                        >
                           Saving Throws
                        </Typography>
                        {/* Saving throw proficiencies by class */}
                        {classID === 'barbarian' && (
                           <>
                              <Typography variant="body2">
                                 • Strength (+
                                 {2 + Math.floor((finalScores.str - 10) / 2)})
                              </Typography>
                              <Typography variant="body2">
                                 • Constitution (+
                                 {2 + Math.floor((finalScores.con - 10) / 2)})
                              </Typography>
                           </>
                        )}
                        {classID === 'fighter' && (
                           <>
                              <Typography variant="body2">
                                 • Strength (+
                                 {2 + Math.floor((finalScores.str - 10) / 2)})
                              </Typography>
                              <Typography variant="body2">
                                 • Constitution (+
                                 {2 + Math.floor((finalScores.con - 10) / 2)})
                              </Typography>
                           </>
                        )}
                        {classID === 'monk' && (
                           <>
                              <Typography variant="body2">
                                 • Strength (+
                                 {2 + Math.floor((finalScores.str - 10) / 2)})
                              </Typography>
                              <Typography variant="body2">
                                 • Dexterity (+
                                 {2 + Math.floor((finalScores.dex - 10) / 2)})
                              </Typography>
                           </>
                        )}
                     </Box>
                  </Box>

                  {/* Class features */}
                  <Box>
                     <Typography
                        variant="subtitle2"
                        sx={{
                           mb: 0.5,
                           textTransform: 'uppercase',
                           letterSpacing: '0.08em',
                        }}
                     >
                        Class Features
                     </Typography>
                     {selectedClass.features.map((f) => (
                        <Typography key={f} variant="body2">
                           • {f}
                        </Typography>
                     ))}
                  </Box>

                  {/* Campaign summary (create mode only) */}
                  {!isJoinMode && selectedCampaign && (
                     <Box>
                        <Divider sx={{ mb: 1.5 }} />
                        <Typography
                           variant="subtitle2"
                           sx={{
                              mb: 0.5,
                              textTransform: 'uppercase',
                              letterSpacing: '0.08em',
                           }}
                        >
                           Authored Campaign
                        </Typography>
                        <Typography variant="body2">
                           {selectedCampaign.title}
                        </Typography>
                        <Typography variant="body2" color="text.secondary">
                           {selectedCampaign.premise}
                        </Typography>
                     </Box>
                  )}

                  {error && <Alert severity="error">{error}</Alert>}

                  <Box
                     sx={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        mt: 1,
                     }}
                  >
                     <Button
                        variant="outlined"
                        onClick={handleBack}
                        disabled={isSubmitting}
                     >
                        Back
                     </Button>
                     <Button
                        variant="contained"
                        onClick={handleSubmit}
                        disabled={isSubmitting}
                        sx={{ minWidth: 180 }}
                     >
                        {isSubmitting
                           ? isJoinMode
                              ? 'Joining...'
                              : 'Creating...'
                           : isJoinMode
                             ? 'Enter the Adventure'
                             : 'Begin Adventure'}
                     </Button>
                  </Box>
               </Box>
            )}
         </Paper>
      </Box>
   );
}

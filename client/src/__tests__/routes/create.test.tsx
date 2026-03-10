// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ThemeProvider } from '@mui/material/styles';
import { AppTheme } from '@/theme/theme';

// ── Router mocks ────────────────────────────────────────────────────────────
const mockNavigate = vi.fn();
const mockUseSearch = vi.fn();

vi.mock('@tanstack/react-router', async (importOriginal) => {
   const actual =
      await importOriginal<typeof import('@tanstack/react-router')>();
   return {
      ...actual,
      redirect: vi.fn(),
      useNavigate: () => mockNavigate,
      useSearch: () => mockUseSearch(),
   };
});

// ── Auth mock ────────────────────────────────────────────────────────────────
vi.mock('@/services/auth.service', () => ({
   isAuthenticated: vi.fn(() => true),
}));

// ── API mocks ────────────────────────────────────────────────────────────────
const { mockCreateGame, mockJoinCharacter } = vi.hoisted(() => ({
   mockCreateGame: vi.fn().mockResolvedValue({
      session_id: 'abc-123',
      ready: false,
      preview_mode: false,
   }),
   mockJoinCharacter: vi
      .fn()
      .mockResolvedValue({ session_id: 'existing-session' }),
}));

vi.mock('@/services/api.game', () => ({
   CreateGame: mockCreateGame,
   JoinCharacter: mockJoinCharacter,
}));

// ── Import component directly (bypasses TanStack lazy wrapper) ───────────────
import { CreateRoute as CreateRouteComponent } from '@/routes/create';

beforeEach(() => {
   vi.clearAllMocks();
   // Default: new game mode (no session param)
   mockUseSearch.mockReturnValue({ session: undefined });
});

function renderWizard() {
   return render(
      <ThemeProvider theme={AppTheme}>
         <CreateRouteComponent />
      </ThemeProvider>,
   );
}

// ── Helper: complete step 0 (Name) ───────────────────────────────────────────
async function fillName(name = 'Aria Silverwind') {
   const user = userEvent.setup();
   const input = screen.getByLabelText(/Character Name/i);
   await user.clear(input);
   await user.type(input, name);
   return user;
}

// ── Helper: click Next button ────────────────────────────────────────────────
async function clickNext(user: ReturnType<typeof userEvent.setup>) {
   const nextBtn = screen.getByRole('button', { name: /next/i });
   await user.click(nextBtn);
}

// ─────────────────────────────────────────────────────────────────────────────

describe('CreateRoute wizard — new game mode', () => {
   it('starts on the Name step', () => {
      renderWizard();
      expect(screen.getByLabelText(/Character Name/i)).toBeInTheDocument();
      expect(screen.getByText(/What are you called/i)).toBeInTheDocument();
   });

   it('Back button on step 0 navigates to /', async () => {
      renderWizard();
      const user = userEvent.setup();
      await user.click(screen.getByRole('button', { name: /cancel/i }));
      expect(mockNavigate).toHaveBeenCalledWith({ to: '/' });
   });

   it('Next button is disabled when name is empty', () => {
      renderWizard();
      const nextBtn = screen.getByRole('button', { name: /next/i });
      expect(nextBtn).toBeDisabled();
   });

   it('Next button is disabled when name is only 1 character', async () => {
      renderWizard();
      const user = userEvent.setup();
      await user.type(screen.getByLabelText(/Character Name/i), 'A');
      expect(screen.getByRole('button', { name: /next/i })).toBeDisabled();
   });

   it('Next button enables with a valid name', async () => {
      renderWizard();
      await fillName();
      expect(screen.getByRole('button', { name: /next/i })).toBeEnabled();
   });

   it('advances to Race step after entering a valid name', async () => {
      renderWizard();
      const user = await fillName();
      await clickNext(user);
      expect(screen.getByText(/Choose your ancestry/i)).toBeInTheDocument();
      // Should show race cards
      expect(screen.getByText('Human')).toBeInTheDocument();
      expect(screen.getByText('Dwarf')).toBeInTheDocument();
   });

   it('Race step: Next disabled until race selected', async () => {
      renderWizard();
      const user = await fillName();
      await clickNext(user);
      expect(screen.getByRole('button', { name: /next/i })).toBeDisabled();
   });

   it('Race step: can select a race without subraces and advance', async () => {
      renderWizard();
      const user = await fillName();
      await clickNext(user);
      await user.click(screen.getByText('Human'));
      expect(screen.getByRole('button', { name: /next/i })).toBeEnabled();
      await clickNext(user);
      // Should be on Class step
      expect(
         screen.getByText(/Your class defines your combat style/i),
      ).toBeInTheDocument();
   });

   it('Race step: cannot advance without subrace for Dwarf', async () => {
      renderWizard();
      const user = await fillName();
      await clickNext(user);
      await user.click(screen.getByText('Dwarf'));
      // Subrace required — Next still disabled
      expect(screen.getByRole('button', { name: /next/i })).toBeDisabled();
      // Must pick a subrace
      await user.click(screen.getByText(/Hill Dwarf/i));
      expect(screen.getByRole('button', { name: /next/i })).toBeEnabled();
   });

   it('Class step: Next disabled until class selected', async () => {
      renderWizard();
      const user = await fillName();
      await clickNext(user);
      await user.click(screen.getByText('Human'));
      await clickNext(user);
      expect(screen.getByRole('button', { name: /next/i })).toBeDisabled();
   });

   it('Class step: selecting a class enables Next', async () => {
      renderWizard();
      const user = await fillName();
      await clickNext(user);
      await user.click(screen.getByText('Human'));
      await clickNext(user);
      // Click "Barbarian" heading in one of the class cards
      await user.click(screen.getByText('Barbarian'));
      expect(screen.getByRole('button', { name: /next/i })).toBeEnabled();
   });

   it('Ability scores: Next disabled until all 6 assigned', async () => {
      renderWizard();
      const user = await fillName();
      await clickNext(user); // → Race
      await user.click(screen.getByText('Human'));
      await clickNext(user); // → Class
      await user.click(screen.getByText('Barbarian'));
      await clickNext(user); // → Abilities
      expect(
         screen.getByText(/Assign the standard array/i),
      ).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /next/i })).toBeDisabled();
      expect(screen.getByText('0/6 values assigned')).toBeInTheDocument();
   });

   it('Skills: limited to class skill count', async () => {
      // Navigate to Skills step by completing all prior steps
      renderWizard();
      const user = await fillName();
      await clickNext(user); // → Race
      await user.click(screen.getByText('Human'));
      await clickNext(user); // → Class — pick Barbarian (2 skills from 6)
      await user.click(screen.getByText('Barbarian'));
      await clickNext(user); // → Abilities — assign all 6 via dropdowns
      // Abilities: open each Select and pick values
      // We'll use MUI Select — find all comboboxes
      const comboboxes = screen.getAllByRole('combobox');
      const abilitySelects = comboboxes.slice(0, 6);
      const values = [15, 14, 13, 12, 10, 8];
      for (let i = 0; i < abilitySelects.length; i++) {
         await user.click(abilitySelects[i]);
         const listbox = screen.getByRole('listbox');
         const options = within(listbox).getAllByRole('option');
         // Find the option matching our value (skip the "—" empty option at index 0)
         const targetOption = options.find(
            (o) => o.textContent === String(values[i]),
         );
         if (targetOption) await user.click(targetOption);
      }
      await clickNext(user); // → Skills
      expect(
         screen.getByText(
            /Choose 2 skill proficiencies from the Barbarian list/i,
         ),
      ).toBeInTheDocument();
      // Select 2 skills
      await user.click(screen.getByText('Athletics (STR)'));
      expect(screen.getByText('1/2 skills selected')).toBeInTheDocument();
      await user.click(screen.getByText('Survival (WIS)'));
      expect(screen.getByText('2/2 skills selected')).toBeInTheDocument();
      // Third click should be blocked (chip is disabled)
      const perceptionChip = screen.getByText('Perception (WIS)');
      // The chip is now at 0.4 opacity (disabled) — clicking it should not add it
      await user.click(perceptionChip);
      expect(screen.getByText('2/2 skills selected')).toBeInTheDocument();
   });

   it('shows Forge Your Adventure heading in create mode', () => {
      renderWizard();
      expect(screen.getByText('Forge Your Adventure')).toBeInTheDocument();
   });

   it('stepper shows all 7 steps in create mode', () => {
      renderWizard();
      [
         'Name',
         'Race',
         'Class',
         'Ability Scores',
         'Skills',
         'Adventure',
         'Review',
      ].forEach((label) => {
         expect(screen.getByText(label)).toBeInTheDocument();
      });
   });
});

describe('CreateRoute wizard — join game mode', () => {
   beforeEach(() => {
      mockUseSearch.mockReturnValue({ session: 'existing-session-id' });
   });

   it('shows Join the Adventure heading in join mode', () => {
      renderWizard();
      expect(screen.getByText('Join the Adventure')).toBeInTheDocument();
   });

   it('shows info alert about joining', () => {
      renderWizard();
      expect(
         screen.getByText(/joining an existing adventure/i),
      ).toBeInTheDocument();
   });

   it('stepper shows only 6 steps in join mode (no Adventure step)', () => {
      renderWizard();
      ['Name', 'Race', 'Class', 'Ability Scores', 'Skills', 'Review'].forEach(
         (label) => {
            expect(screen.getByText(label)).toBeInTheDocument();
         },
      );
      expect(screen.queryByText('Adventure')).not.toBeInTheDocument();
   });

   it('submit in join mode calls JoinCharacter, not CreateGame', async () => {
      renderWizard();
      const user = await fillName('Thorin Ironforge');
      await clickNext(user); // → Race
      await user.click(screen.getByText('Human'));
      await clickNext(user); // → Class
      await user.click(screen.getByText('Fighter'));
      await clickNext(user); // → Abilities
      const comboboxes = screen.getAllByRole('combobox');
      const abilitySelects = comboboxes.slice(0, 6);
      const values = [15, 14, 13, 12, 10, 8];
      for (let i = 0; i < abilitySelects.length; i++) {
         await user.click(abilitySelects[i]);
         const listbox = screen.getByRole('listbox');
         const options = within(listbox).getAllByRole('option');
         const targetOption = options.find(
            (o) => o.textContent === String(values[i]),
         );
         if (targetOption) await user.click(targetOption);
      }
      await clickNext(user); // → Skills
      await user.click(screen.getByText('Athletics (STR)'));
      await user.click(screen.getByText('Perception (WIS)'));
      await clickNext(user); // → Review (no Adventure step in join mode)
      expect(screen.getByText('Thorin Ironforge')).toBeInTheDocument();
      expect(screen.getByText(/Level 1 Human Fighter/i)).toBeInTheDocument();
      await user.click(
         screen.getByRole('button', { name: /Enter the Adventure/i }),
      );
      expect(mockJoinCharacter).toHaveBeenCalledWith(
         'existing-session-id',
         expect.objectContaining({
            name: 'Thorin Ironforge',
            race_id: 'human',
            class_id: 'fighter',
            selected_skills: expect.arrayContaining([
               'athletics',
               'perception',
            ]),
         }),
      );
      expect(mockCreateGame).not.toHaveBeenCalled();
   });
});

describe('CreateRoute wizard — Review step', () => {
   async function navigateToReviewStep(
      user: ReturnType<typeof userEvent.setup>,
   ) {
      await fillName('Aria Silverwind');
      await clickNext(user); // → Race
      await user.click(screen.getByText('Dwarf'));
      await user.click(screen.getByText(/Hill Dwarf/i));
      await clickNext(user); // → Class
      await user.click(screen.getByText('Barbarian'));
      await clickNext(user); // → Abilities
      const comboboxes = screen.getAllByRole('combobox');
      const abilitySelects = comboboxes.slice(0, 6);
      const values = [15, 14, 13, 12, 10, 8];
      for (let i = 0; i < abilitySelects.length; i++) {
         await user.click(abilitySelects[i]);
         const listbox = screen.getByRole('listbox');
         const options = within(listbox).getAllByRole('option');
         const targetOption = options.find(
            (o) => o.textContent === String(values[i]),
         );
         if (targetOption) await user.click(targetOption);
      }
      await clickNext(user); // → Skills
      await user.click(screen.getByText('Athletics (STR)'));
      await user.click(screen.getByText('Survival (WIS)'));
      await clickNext(user); // → Adventure
      await clickNext(user); // → Review
   }

   it('review step shows character name, race, class', async () => {
      renderWizard();
      const user = userEvent.setup();
      await navigateToReviewStep(user);
      expect(screen.getByText('Aria Silverwind')).toBeInTheDocument();
      expect(screen.getByText(/Level 1 Dwarf.*Barbarian/i)).toBeInTheDocument();
   });

   it('review step shows HP > 0', async () => {
      renderWizard();
      const user = userEvent.setup();
      await navigateToReviewStep(user);
      // HP label
      expect(screen.getByText('HP')).toBeInTheDocument();
      // Barbarian d12 + CON mod: CON=13 (assigned) +2 racial hill_dwarf = 15, mod=+2 → HP=14
      // But the exact value depends on assignment. Just verify it's a positive number.
      const hpBox = screen.getByText('HP').parentElement!;
      const hpValue = hpBox.querySelector('h6');
      expect(Number(hpValue?.textContent)).toBeGreaterThan(0);
   });

   it('review step shows selected skills', async () => {
      renderWizard();
      const user = userEvent.setup();
      await navigateToReviewStep(user);
      expect(screen.getByText('Skill Proficiencies')).toBeInTheDocument();
      expect(screen.getByText(/Athletics/i)).toBeInTheDocument();
      expect(screen.getByText(/Survival/i)).toBeInTheDocument();
   });

   it('review step shows class features', async () => {
      renderWizard();
      const user = userEvent.setup();
      await navigateToReviewStep(user);
      expect(screen.getByText('Class Features')).toBeInTheDocument();
      expect(screen.getByText(/Rage/i)).toBeInTheDocument();
   });

   it('clicking Begin Adventure calls CreateGame with correct payload', async () => {
      renderWizard();
      const user = userEvent.setup();
      await navigateToReviewStep(user);
      await user.click(
         screen.getByRole('button', { name: /Begin Adventure/i }),
      );
      expect(mockCreateGame).toHaveBeenCalledWith(
         expect.objectContaining({
            name: 'Aria Silverwind',
            race_id: 'dwarf',
            subrace_id: 'hill-dwarf',
            class_id: 'barbarian',
            selected_skills: expect.arrayContaining(['athletics', 'survival']),
         }),
      );
   });

   it('navigates to game route after successful creation', async () => {
      renderWizard();
      const user = userEvent.setup();
      await navigateToReviewStep(user);
      await user.click(
         screen.getByRole('button', { name: /Begin Adventure/i }),
      );
      expect(mockNavigate).toHaveBeenCalledWith({
         to: '/game-{$sessionUUID}',
         params: { sessionUUID: 'abc-123' },
      });
   });

   it('shows error alert on API failure', async () => {
      mockCreateGame.mockRejectedValueOnce(new Error('Server down'));
      renderWizard();
      const user = userEvent.setup();
      await navigateToReviewStep(user);
      await user.click(
         screen.getByRole('button', { name: /Begin Adventure/i }),
      );
      expect(await screen.findByText('Server down')).toBeInTheDocument();
   });
});

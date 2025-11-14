import { createTheme } from "@mui/material/styles";

// Dark Fantasy Theme - Medieval dungeon aesthetic
export const AppTheme = createTheme({
  cssVariables: { colorSchemeSelector: "class" },
  colorSchemes: {
    dark: {
      palette: {
        primary: {
          main: "#C9A962", // Gold
          light: "#FFD700",
          dark: "#8B7355",
          contrastText: "#1a0a2e",
        },
        secondary: {
          main: "#6B4E9D", // Deep purple
          light: "#9575CD",
          dark: "#3E2C5E",
          contrastText: "#ffffff",
        },
        error: { main: "#D32F2F" },
        warning: { main: "#F57C00" },
        info: { main: "#4A90E2" },
        success: { main: "#388E3C" },
        background: {
          default: "#0d0508", // Very dark purple-black
          paper: "#1a0f1e", // Dark purple
        },
        divider: "#3e2c2e",
        text: {
          primary: "#E8DCC4", // Parchment color
          secondary: "#B8A588",
          disabled: "#6D5D4B",
        },
      },
    },
    light: {
      palette: {
        primary: {
          main: "#8B6F47", // RuneScape brown/tan
          light: "#A0826D",
          dark: "#6B5638",
          contrastText: "#2C1810",
        },
        secondary: {
          main: "#654321", // Dark chocolate brown
          light: "#8B6F47",
          dark: "#3E2723",
          contrastText: "#E8DCC4",
        },
        error: { main: "#8B0000" },
        warning: { main: "#D4A574" },
        info: { main: "#6B8E23" },
        success: { main: "#556B2F" },
        background: {
          default: "#E8DCC4", // Papyrus/scroll color
          paper: "#D4C5A9", // Darker papyrus for panels
        },
        divider: "#A0826D",
        text: {
          primary: "#2C1810", // Very dark brown
          secondary: "#5D4037", // Medium brown
          disabled: "#8B7355",
        },
      },
    },
  },
  typography: {
    fontFamily: '"Cinzel", "Georgia", "Times New Roman", serif',
    fontSize: 18,
    h1: {
      fontFamily: '"Cinzel Decorative", "Georgia", serif',
      fontWeight: 700,
      letterSpacing: '0.02em',
      fontSize: '3.5rem',
    },
    h2: {
      fontFamily: '"Cinzel Decorative", "Georgia", serif',
      fontWeight: 600,
      letterSpacing: '0.02em',
      fontSize: '3rem',
    },
    h3: {
      fontFamily: '"Cinzel", "Georgia", serif',
      fontWeight: 600,
      fontSize: '2.5rem',
    },
    h4: {
      fontFamily: '"Cinzel", "Georgia", serif',
      fontWeight: 600,
      fontSize: '2rem',
    },
    h5: {
      fontFamily: '"Cinzel", "Georgia", serif',
      fontWeight: 500,
      fontSize: '1.75rem',
    },
    h6: {
      fontFamily: '"Cinzel", "Georgia", serif',
      fontWeight: 500,
      fontSize: '1.5rem',
    },
    body1: {
      fontFamily: '"Crimson Text", "Georgia", serif',
      fontSize: '1.25rem',
      lineHeight: 1.7,
    },
    body2: {
      fontFamily: '"Crimson Text", "Georgia", serif',
      fontSize: '1.125rem',
      lineHeight: 1.6,
    },
    button: {
      fontFamily: '"Cinzel", "Georgia", serif',
      fontWeight: 600,
      letterSpacing: '0.05em',
      textTransform: 'uppercase',
      fontSize: '1.125rem',
    },
  },
  shape: { borderRadius: 4 },
  components: {
    MuiPaper: {
      styleOverrides: {
        root: {
          backgroundImage: 'linear-gradient(rgba(106, 78, 157, 0.05), rgba(201, 169, 98, 0.05))',
          border: '1px solid rgba(201, 169, 98, 0.2)',
          boxShadow: '0 4px 20px rgba(0, 0, 0, 0.5), inset 0 1px 0 rgba(201, 169, 98, 0.1)',
        },
      },
    },
    MuiButton: {
      styleOverrides: {
        root: {
          borderRadius: 4,
          padding: '8px 20px',
        },
        contained: {
          '&:hover': {
            boxShadow: '0 4px 12px rgba(201, 169, 98, 0.6), inset 0 1px 0 rgba(255, 215, 0, 0.5)',
          },
        },
      },
    },
    MuiChip: {
      styleOverrides: {
        root: {
          borderRadius: 4,
          fontFamily: '"Cinzel", "Georgia", serif',
          fontSize: '0.75rem',
          fontWeight: 600,
        },
      },
    },
  },
});

// UI Color Tokens - use these instead of hardcoded colors
export const ColorTokens = {
  dark: {
    text: {
      primary: "#E8DCC4",   // Parchment
      secondary: "#C9A962",  // Gold (better contrast for legend)
      muted: "#6D5D4B",     // Disabled
    },
    icon: "#C9A962",         // Icon buttons (gold for visibility)
    accent: "#FFA726",       // Stairs/torch icon (orange)
    chipOutline: "rgba(201, 169, 98, 0.5)",  // Chip border
    chipText: "#E8DCC4",     // Chip outlined text
  },
  light: {
    text: {
      primary: "#2C1810",   // Very dark brown
      secondary: "#5D4037",  // Medium brown
      muted: "#8B7355",     // Disabled
    },
    icon: "#3E2723",         // Icon buttons (dark chocolate)
    accent: "#D84315",       // Stairs/torch icon (darker red-orange for visibility)
    chipOutline: "#8B6F47",  // Chip border (brown)
    chipText: "#2C1810",     // Chip outlined text
  },
};

// Helper to get mode-specific colors
export const getColorByMode = (mode: 'light' | 'dark', category: keyof typeof ColorTokens.dark) => {
  return ColorTokens[mode][category];
};

// Custom dungeon colors for map (canvas rendering)
export const DungeonColors = {
  wall: "#3E2723",
  wallHighlight: "#5D4037",
  floor: "#4E342E",
  corridor: "#6D4C41",
  door: "#8B7355",
  doorway: "#A1887F",
  currentRoom: "#C9A962",
  adjacentRoom: "#6B4E9D",
  exploredRoom: "#5D4037",
  unexploredRoom: "#2C1810",
  torch: "#FFA726",
  torchGlow: "rgba(255, 167, 38, 0.3)",
  fog: "rgba(13, 5, 8, 0.6)",
};

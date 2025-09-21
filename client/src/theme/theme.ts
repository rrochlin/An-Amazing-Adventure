import { createTheme } from "@mui/material/styles";

export const AppTheme = createTheme({
  cssVariables: { colorSchemeSelector: "class" },
  colorSchemes: {
    light: {
      palette: {
        mode: "light",
        primary: {
          main: "#1976d2",
          light: "#63a4ff",
          dark: "#004ba0",
          contrastText: "#ffffff",
        },
        secondary: {
          main: "#9c27b0",
          light: "#d05ce3",
          dark: "#6a0080",
          contrastText: "#ffffff",
        },
        error: { main: "#d32f2f" },
        warning: { main: "#ed6c02" },
        info: { main: "#0288d1" },
        success: { main: "#2e7d32" },
        background: {
          default: "#f5f7fb",
          paper: "#ffffff",
        },
        divider: "#e0e3e7",
        text: {
          primary: "rgba(0,0,0,0.87)",
          secondary: "rgba(0,0,0,0.6)",
          disabled: "rgba(0,0,0,0.38)",
        },
      },
    },
    dark: {
      palette: {
        mode: "dark",
        primary: {
          main: "#90caf9",
          light: "#c3fdff",
          dark: "#5d99c6",
          contrastText: "#0b0f19",
        },
        secondary: {
          main: "#ce93d8",
          light: "#ffc4ff",
          dark: "#9c64a6",
          contrastText: "#0b0f19",
        },
        error: { main: "#f44336" },
        warning: { main: "#ff9800" },
        info: { main: "#29b6f6" },
        success: { main: "#66bb6a" },
        background: {
          default: "#0b0f19",
          paper: "#121826",
        },
        divider: "#2a3350",
        text: {
          primary: "rgba(255,255,255,0.87)",
          secondary: "rgba(255,255,255,0.6)",
          disabled: "rgba(255,255,255,0.38)",
        },
      },
    },
  },
  shape: { borderRadius: 8 },
});

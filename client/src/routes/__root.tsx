import { Outlet, createRootRoute } from "@tanstack/react-router";
import { TanStackRouterDevtoolsPanel } from "@tanstack/react-router-devtools";
import { TanstackDevtools } from "@tanstack/react-devtools";
import { Header } from "@/components/Header";
import { ThemeProvider } from "@mui/material/styles";
import { CssBaseline, GlobalStyles, Box } from "@mui/material";
import { AppTheme } from "@/theme/theme";

export const Route = createRootRoute({
  component: () => (
    <ThemeProvider theme={AppTheme}>
      <CssBaseline />
      <GlobalStyles
        styles={(theme) => ({
          body: {
            backgroundImage:
              theme.palette.mode === "dark"
                ? `
                  radial-gradient(circle at 20% 50%, rgba(106, 78, 157, 0.05) 0%, transparent 50%),
                  radial-gradient(circle at 80% 80%, rgba(201, 169, 98, 0.05) 0%, transparent 50%),
                  linear-gradient(180deg, rgba(13, 5, 8, 1) 0%, rgba(26, 15, 30, 1) 100%)
                `
                : `
                  radial-gradient(circle at 20% 50%, rgba(139, 111, 71, 0.1) 0%, transparent 50%),
                  radial-gradient(circle at 80% 80%, rgba(160, 130, 109, 0.1) 0%, transparent 50%),
                  linear-gradient(180deg, #E8DCC4 0%, #D4C5A9 100%)
                `,
            backgroundAttachment: "fixed",
          },
          "*::selection": {
            backgroundColor: theme.palette.primary.main,
            color: theme.palette.primary.contrastText,
          },
        })}
      />
      <Box
        sx={(theme) => ({
          minHeight: "100vh",
          position: "relative",
          "&::before": {
            content: '""',
            position: "fixed",
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundImage:
              theme.palette.mode === "dark"
                ? "url(\"data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%23C9A962' fill-opacity='0.02'%3E%3Cpath d='M36 34v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zm0-30V0h-2v4h-4v2h4v4h2V6h4V4h-4zM6 34v-4H4v4H0v2h4v4h2v-4h4v-2H6zM6 4V0H4v4H0v2h4v4h2V6h4V4H6z'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E\")"
                : "url(\"data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%236B5638' fill-opacity='0.08'%3E%3Cpath d='M36 34v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zm0-30V0h-2v4h-4v2h4v4h2V6h4V4h-4zM6 34v-4H4v4H0v2h4v4h2v-4h4v-2H6zM6 4V0H4v4H0v2h4v4h2V6h4V4H6z'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E\")",
            opacity: theme.palette.mode === "dark" ? 0.3 : 0.25,
            pointerEvents: "none",
            zIndex: 0,
          },
        })}
      >
        <Header />
        <Outlet />

        <TanstackDevtools
          config={{
            position: "bottom-left",
          }}
          plugins={[
            {
              name: "Tanstack Router",
              render: <TanStackRouterDevtoolsPanel />,
            },
          ]}
        />
      </Box>
    </ThemeProvider>
  ),
});

import { Outlet, createRootRoute } from "@tanstack/react-router";
import { TanStackRouterDevtoolsPanel } from "@tanstack/react-router-devtools";
import { TanstackDevtools } from "@tanstack/react-devtools";
import { Header } from "@/components/Header";
import { ThemeProvider } from "@mui/material/styles";
import { AppTheme } from "@/theme/theme";

export const Route = createRootRoute({
  component: () => (
    <ThemeProvider theme={AppTheme}>
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
    </ThemeProvider>
  ),
});

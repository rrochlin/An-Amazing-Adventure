import { AppBar, Box, Toolbar, css, styled } from "@mui/material";
import { CustomLink } from "./CustomLink";
import AccountPanel from "./AccountPanel";

const StyledCustomLink = styled(CustomLink)(
  ({ theme }) => css`
    color: ${theme.palette.common.white};
  `,
);

export function Header() {
  return (
    <Box sx={{ flexGrow: 1 }}>
      <AppBar position="static">
        <Toolbar>
          <div style={{ flex: 1 }}>
            <StyledCustomLink to="/">Home</StyledCustomLink>
          </div>
          <AccountPanel />
        </Toolbar>
      </AppBar>
    </Box>
  );
}

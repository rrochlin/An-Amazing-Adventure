import { ClearUserAuth, isAuthenticated } from "@/services/auth.service";
import { useNavigate } from "@tanstack/react-router";
import React, { useState, type ReactElement } from "react";
import AccountCircleIcon from "@mui/icons-material/AccountCircle";
import {
  Collapse,
  IconButton,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
  useColorScheme,
} from "@mui/material";
import LogoutIcon from "@mui/icons-material/Logout";
import DarkModeIcon from "@mui/icons-material/DarkMode";
import LightModeIcon from "@mui/icons-material/LightMode";
import SettingsBrightnessIcon from "@mui/icons-material/SettingsBrightness";
import MonitorIcon from "@mui/icons-material/Monitor";
import LoginIcon from "@mui/icons-material/Login";
import AccountBoxIcon from "@mui/icons-material/AccountBox";

export default function AccountPanel() {
  const navigate = useNavigate();
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const [open, setOpen] = useState(false);
  const { setMode } = useColorScheme();

  const handleMenu = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  const handleThemeChange = () => {
    setOpen((prev) => !prev);
  };

  const handleProfile = () => {
    navigate({
      to: "/profile",
    });
  };

  const handleLogout = () => {
    ClearUserAuth();
    navigate({
      to: "/login",
    });
  };

  const handleSignIn = () => {
    navigate({
      to: "/login",
      search: { redirect: location.href },
    });
  };
  interface themeOpt {
    mode: "dark" | "light" | "system";
    msg: string;
    icon: ReactElement;
  }
  const opts: themeOpt[] = [
    { mode: "dark", msg: "Dark Mode", icon: <DarkModeIcon /> },
    { mode: "light", msg: "Light Mode", icon: <LightModeIcon /> },
    {
      mode: "system",
      msg: "System Settings",
      icon: <SettingsBrightnessIcon />,
    },
  ];

  return (
    <>
      <IconButton onClick={handleMenu}>
        <AccountBoxIcon />
      </IconButton>
      <Menu
        id="menu-appbar"
        anchorEl={anchorEl}
        keepMounted
        open={Boolean(anchorEl)}
        onClose={handleClose}
      >
        <MenuItem onClick={handleProfile}>
          <ListItemIcon>
            <AccountCircleIcon />
          </ListItemIcon>
          <ListItemText primary="Profile" />
        </MenuItem>

        {/* Theme changing */}
        <MenuItem onClick={handleThemeChange}>
          <ListItemIcon>
            <MonitorIcon />
          </ListItemIcon>
          <ListItemText primary={"Select Theme"} />
        </MenuItem>
        <Collapse in={open} timeout="auto" unmountOnExit>
          <List component="div" disablePadding>
            {opts.map((opt: themeOpt) => (
              <ListItemButton onClick={() => setMode(opt.mode)} key={opt.mode}>
                <ListItemIcon>{opt.icon}</ListItemIcon>
                <ListItemText primary={opt.msg} />
              </ListItemButton>
            ))}
          </List>
        </Collapse>

        {/* Sign in/Sign out */}
        {isAuthenticated() ? (
          <MenuItem onClick={handleLogout}>
            <ListItemIcon>
              <LogoutIcon />
            </ListItemIcon>
            <ListItemText primary="Sign Out" />
          </MenuItem>
        ) : (
          <MenuItem onClick={handleSignIn}>
            <ListItemIcon>
              <LoginIcon />
            </ListItemIcon>
            <ListItemText primary="Sign In" />
          </MenuItem>
        )}
      </Menu>
    </>
  );
}

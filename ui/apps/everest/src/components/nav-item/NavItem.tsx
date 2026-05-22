import {
  Box,
  IconButton,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Tooltip,
} from '@mui/material';
import React from 'react';
import { NavLink, useNavigate } from 'react-router-dom';
import { NavItemProps } from './NavItem.types';

export const NavItem = ({
  open,
  icon,
  text,
  to,
  onClick,
  ...listItemProps
}: NavItemProps) => {
  const navigate = useNavigate();

  const redirect = (redirectUrl: string) => {
    navigate(redirectUrl);
    onClick();
  };

  return (
    <ListItem disablePadding sx={{ display: 'block' }} {...listItemProps}>
      {open ? (
        <ListItemButton
          component={NavLink}
          to={to}
          sx={{
            minHeight: 48,
            justifyContent: 'initial',
            px: 2.5,
          }}
          onClick={onClick}
        >
          <ListItemIcon
            sx={{
              minWidth: 0,
              mr: 3,
              justifyContent: 'center',
            }}
          >
            {typeof icon === 'string' ? (
              <img src={icon} alt="" width={24} height={24} />
            ) : (
              React.createElement(icon)
            )}
          </ListItemIcon>

          <ListItemText primary={text} />
        </ListItemButton>
      ) : (
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            minHeight: 48,
            px: '12px',
          }}
        >
          <Tooltip title={text} placement="right" arrow>
            <IconButton onClick={() => redirect(to)}>
              {typeof icon === 'string' ? (
                <img src={icon} alt="" width={24} height={24} />
              ) : (
                React.createElement(icon)
              )}
            </IconButton>
          </Tooltip>
        </Box>
      )}
    </ListItem>
  );
};

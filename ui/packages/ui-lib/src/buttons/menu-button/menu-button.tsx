import React, { useState } from 'react';
import { Button, Menu } from '@mui/material';
import ArrowDropDownOutlinedIcon from '@mui/icons-material/ArrowDropDownOutlined';
import { MenuButtonProps } from './menu-button.types';

const MenuButton = ({
  children,
  buttonText,
  buttonProps,
  menuProps,
  matchAnchorWidth,
}: MenuButtonProps) => {
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const open = !!anchorEl;

  const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  return (
    <>
      <Button
        id="menu-button"
        data-testid="menu-button"
        aria-controls={open ? 'menu-button-dropdown' : undefined}
        aria-haspopup="true"
        aria-expanded={open ? 'true' : undefined}
        onClick={handleClick}
        endIcon={<ArrowDropDownOutlinedIcon />}
        size="small"
        variant="contained"
        {...buttonProps}
      >
        {buttonText}
      </Button>
      <Menu
        id="menu-button-dropdown"
        anchorEl={anchorEl}
        open={open}
        onClose={handleClose}
        MenuListProps={{
          'aria-labelledby': 'menu-button',
          ...menuProps?.MenuListProps,
        }}
        PaperProps={{
          ...(matchAnchorWidth && anchorEl
            ? { style: { minWidth: anchorEl.offsetWidth } }
            : {}),
          ...menuProps?.PaperProps,
        }}
        {...menuProps}
      >
        {children && children(handleClose)}
      </Menu>
    </>
  );
};

export default MenuButton;

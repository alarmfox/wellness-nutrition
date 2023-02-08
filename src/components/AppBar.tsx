import * as React from 'react';
import { AppBar, Container, Toolbar, Box, IconButton, Menu, MenuItem, Typography, Button, Tooltip } from '@mui/material';
import MenuIcon from '@mui/icons-material/Menu';
import { signOut } from 'next-auth/react';
import { AccountCircle } from '@mui/icons-material';

type Page = {
  name: string;
  id: number;
  path: string;
}
 

const pages: Page[]= [
  {
    name: 'Home',
    id: 1,
    path: '/',
  },
]

type Action = {
  name: string;
  id: number;
  callback: () => void
};

const actions: Action[] = [
  {
    id: 1,
    name: 'Esci',
    // eslint-disable-next-line @typescript-eslint/no-misused-promises
    callback: () => signOut({ callbackUrl: '/'})
  }
];

export function ResponsiveAppBar() {

  const [anchorElNav, setAnchorElNav] = React.useState<null | HTMLElement>(null);
  const [anchorElUser, setAnchorElUser] = React.useState<null | HTMLElement>(null);

  const handleOpenNavMenu = React.useCallback((event: React.MouseEvent<HTMLElement>) => {
    setAnchorElNav(event.currentTarget);
  }, []);

  const handleOpenUserMenu = React.useCallback((event: React.MouseEvent<HTMLElement>) => {
    setAnchorElUser(event.currentTarget);
  }, []);

  const handleCloseNavMenu = React.useCallback(() => {
    setAnchorElNav(null);
  }, []);

  const handleCloseUserMenu = React.useCallback(() => {
    setAnchorElUser(null);
  },[]);

  return (
    <AppBar position="static">
      <Container maxWidth="xl">
        <Toolbar disableGutters>
          <Box sx={{ flexGrow: 1, display: { xs: 'flex', md: 'none' } }}>
            <IconButton
              size="large"
              aria-label="account of current user"
              aria-controls="menu-appbar"
              aria-haspopup="true"
              onClick={handleOpenNavMenu}
              color="inherit"
            >
              <MenuIcon />
            </IconButton>
            <Menu
              id="menu-appbar"
              anchorEl={anchorElNav}
              anchorOrigin={{
                vertical: 'bottom',
                horizontal: 'left',
              }}
              keepMounted
              transformOrigin={{
                vertical: 'top',
                horizontal: 'left',
              }}
              open={Boolean(anchorElNav)}
              onClose={handleCloseNavMenu}
              sx={{
                display: { xs: 'block', md: 'none' },
              }}
            >
              {pages.map((page) => (
                <MenuItem component="a" href={page.path} key={page.id}>
                  <Typography textAlign="center">{page.name}</Typography>
                </MenuItem>
              ))}
            </Menu>
          </Box>
          <Typography
            variant="body1"
            noWrap
            component="a"
            href=""
            sx={{
              mr: 2,
              display: { xs: 'flex', md: 'none' },
              flexGrow: 1,
              fontFamily: 'monospace',
              fontWeight: 500,
              letterSpacing: '.1rem',
              color: 'inherit',
              textDecoration: 'none',
            }}
          >
            Wellness & Nutrition
          </Typography>
          <Box sx={{ flexGrow: 1, display: { xs: 'none', md: 'flex' } }}>
            {pages.map((page) => (
              <Button
                key={page.id}
                href={page.path}
                sx={{ my: 2, color: 'white', display: 'block' }}
              >
                {page.name}
              </Button>
            ))}
          </Box>

          <Box sx={{ flexGrow: 0 }}>
            <Tooltip title="Open settings">
              <IconButton onClick={handleOpenUserMenu} sx={{ p: 0 }}>
                <AccountCircle />
              </IconButton>
            </Tooltip>
            <Menu
              sx={{ mt: '45px' }}
              id="menu-appbar"
              anchorEl={anchorElUser}
              anchorOrigin={{
                vertical: 'top',
                horizontal: 'right',
              }}
              
              keepMounted
              transformOrigin={{
                vertical: 'top',
                horizontal: 'right',
              }}
              open={Boolean(anchorElUser)}
              onClose={handleCloseUserMenu}
            >
              {actions.map((action) => (
                // eslint-disable-next-line @typescript-eslint/no-misused-promises, @typescript-eslint/no-unsafe-assignment
                <MenuItem key={action.id} onClick={action.callback}>
                  <Typography textAlign="center">{action.name}</Typography>
                </MenuItem>
              ))}
            </Menu>
          </Box>
        </Toolbar>
      </Container>
    </AppBar>
  );
}
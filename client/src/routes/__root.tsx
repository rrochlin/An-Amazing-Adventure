/// <reference types="vite/client" />
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools'
import {
	HeadContent,
	Outlet,
	Scripts,
	createRootRoute,
} from '@tanstack/react-router'
import { CacheProvider } from '@emotion/react'
import { Container, CssBaseline, StyledEngineProvider } from '@mui/material'
import createCache from '@emotion/cache'
import fontsourceVariableRobotoCss from '@fontsource-variable/roboto?url'
import React from 'react'
import { theme } from '~/setup/theme'
import { Header } from '~/components/Header'
import { AppProvider } from '@toolpad/core/AppProvider'

export const Route = createRootRoute({
	head: () => ({
		links: [{ rel: 'stylesheet', href: fontsourceVariableRobotoCss }],
	}),
	component: RootComponent,
})

function RootComponent() {
	return (
		<RootDocument>
			<Outlet />
		</RootDocument>
	)
}

function Providers({ children }: { children: React.ReactNode }) {
	const emotionCache = createCache({ key: 'css' })

	return (
		<StyledEngineProvider injectFirst>
			<CacheProvider value={emotionCache}>
				<AppProvider theme={theme}>
					<CssBaseline />
					{children}
				</AppProvider>
			</CacheProvider>
		</StyledEngineProvider>
	)
}

function RootDocument({ children }: { children: React.ReactNode }) {
	return (
		<html>
			<head>
				<HeadContent />
			</head>
			<body>
				<Providers>
					<Header />

					<Container component="main" sx={{ paddingBlock: 4 }}>
						{children}
					</Container>
				</Providers>

				<TanStackRouterDevtools position="bottom-right" />
				<Scripts />
			</body>
		</html>
	)
}

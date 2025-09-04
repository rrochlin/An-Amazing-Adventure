import { tanstackStart } from '@tanstack/react-start/plugin/vite'
import { defineConfig } from 'vite'
import tsConfigPaths from 'vite-tsconfig-paths'
import react from '@vitejs/plugin-react'

export default defineConfig({
	server: {
		port: 5173,
	},
	ssr: {
		noExternal: ['@mui/*'],
	},
	plugins: [
		tsConfigPaths({
			projects: ['./tsconfig.json'],
		}),
		tanstackStart(),
	],
})

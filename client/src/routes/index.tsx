import { createFileRoute, Link, linkOptions } from '@tanstack/react-router'
import { Stack, Typography } from '@mui/material'
import z from 'zod'

export const Route = createFileRoute('/')({
	validateSearch: z.object({
		count: z.number().optional(),
	}),
	component: RouteComponent,
})

function RouteComponent() {
	const loginOptions = linkOptions({
		to: '/login',
	})
	return (
		<Stack alignItems="center">
			<Typography variant="h1" marginBlockEnd={4}>
				<Link {...loginOptions} >Login</Link>
			</Typography>
		</Stack>
	)
}

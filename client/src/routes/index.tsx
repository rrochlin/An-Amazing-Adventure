import { createFileRoute, Link, linkOptions, Navigate, redirect, useLoaderData } from '@tanstack/react-router'
import { Button, Dialog, DialogActions, DialogContent, DialogContentText, DialogTitle, List, ListItem, ListItemButton, ListItemIcon, ListItemText, ListSubheader, Stack, TextField, Typography } from '@mui/material'
import z from 'zod'
import { isAuthenticated } from '@/services/auth.service'
import { ListGames, StartGame } from '@/services/api.game'
import AddIcon from '@mui/icons-material/Add';
import { useState } from 'react'
import type { ListGamesResponse } from '@/types/api.types'

export const Route = createFileRoute('/')({
	validateSearch: z.object({
		count: z.number().optional(),
	}),
	component: RouteComponent,
	beforeLoad: (() => {
		if (!isAuthenticated()) {
			throw redirect({
				to: '/login',
				search: { redirect: location.href }
			})
		}
	}),
	loader: (async () => {
		const games = await ListGames()
		return games
	})
})

function RouteComponent() {
	const [games, setGames] = useState<ListGamesResponse[]>(Route.useLoaderData())
	const [open, setOpen] = useState(false);
	const loginOptions = linkOptions({
		to: '/login',
	})

	const handleClickOpen = () => {
		setOpen(true);
	};

	const handleClose = () => {
		setOpen(false);
	};

	const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
		event.preventDefault();
		const formData = new FormData(event.currentTarget);
		const formJson = Object.fromEntries((formData as any).entries());
		const name = formJson.characterName;
		const response = await StartGame({ playerName: name })
		handleClose();
		if (!response.success) {
			console.error(response.error)
			alert("Error creating game please try again later")
			return
		}
		setGames(prev => [...prev, { playerName: name, sessionId: response.sessionUUID }])

	}


	return (
		<>
			<Dialog open={open} onClose={handleClose}>
				<DialogTitle>Subscribe</DialogTitle>
				<DialogContent>
					<DialogContentText>
						Enter a name for your adventurer to begin questing!
					</DialogContentText>
					<form onSubmit={handleSubmit} id="game-form">
						<TextField
							autoFocus
							required
							margin="dense"
							id="name"
							name="characterName"
							label="Character Name"
							type="text"
							fullWidth
							variant="standard" />
					</form>
				</DialogContent>
				<DialogActions>
					<Button onClick={handleClose}>Cancel</Button>
					<Button type="submit" form="game-form">
						Create World
					</Button>
				</DialogActions>
			</Dialog>
			<Stack alignItems="center">
				<List
					sx={{ width: '100%', maxWidth: 360, bgcolor: 'background.paper' }}
					component="nav"
					aria-labelledby="nested-list-subheader"
					subheader={<ListSubheader component="div" id="nested-list-subheader">
						Games by character name
					</ListSubheader>}
				>
					{games.map((game: ListGamesResponse) => {
						return (
							<ListItemButton onClick={() => Navigate({
								to: '/game-{$sessionUUID}',
								params: { sessionUUID: game.sessionId }
							})}>
								<ListItemText primary={game.playerName} />
							</ListItemButton>
						)
					})}
					<ListItemButton onClick={handleClickOpen}>
						<ListItemIcon><AddIcon /></ListItemIcon>
						<ListItemText primary={"Create a new Game"} />
					</ListItemButton>

				</List>

			</Stack>
		</>
	)
}

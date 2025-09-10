import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { type AuthProvider } from '@toolpad/core/SignInPage'
import { Login } from '../services/api.users';
import { TextField, Button, Box, Paper, Typography } from '@mui/material';
import { useState } from 'react';
import { success } from 'zod';


export const Route = createFileRoute('/login')({
	component: RouteComponent,
})

const signIn: (provider: AuthProvider, formData: FormData) => Promise<boolean> = async (
	provider,
	formData,
) => {
	const login = await Login({
		email: formData.get('email')?.toString() || "",
		password: formData.get('password')?.toString() || "",
	})
	if (!login.success) {
		alert("incorrect credentials")
		return false
	}
	return true
};

function RouteComponent() {
	const [email, setEmail] = useState('');
	const [password, setPassword] = useState('');
	const navigate = useNavigate()

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		const formData = new FormData();
		formData.set('email', email);
		formData.set('password', password);
		if (!(await signIn({ id: 'credentials', name: 'Email and Password' } as AuthProvider, formData))) return;
		navigate({ to: '/' })
	};

	return (
		<Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '50vh' }}>
			<Paper sx={{ p: 4, maxWidth: 400, width: '100%' }}>
				<Typography variant="h4" component="h1" gutterBottom align="center">
					Sign In
				</Typography>
				<form onSubmit={handleSubmit}>
					<TextField
						fullWidth
						placeholder="Email"
						type="email"
						value={email}
						onChange={(e) => setEmail(e.target.value)}
						margin="normal"
						required
						autoComplete="email"
					/>
					<TextField
						fullWidth
						placeholder="Password"
						type="password"
						value={password}
						onChange={(e) => setPassword(e.target.value)}
						margin="normal"
						required
						autoComplete="current-password"
					/>
					<Button
						type="submit"
						fullWidth
						variant="contained"
						sx={{ mt: 3, mb: 2 }}
					>
						Sign In
					</Button>
				</form>

				<Typography variant="body2" align="center">
					Don't have an account?{' '}
					<Link to="/signup" style={{ color: '#1976d2', textDecoration: 'none' }}>
						Sign Up
					</Link>
				</Typography>
			</Paper>
		</Box>
	)
}

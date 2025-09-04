import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { TextField, Button, Box, Paper, Typography, Alert } from '@mui/material';
import { useState } from 'react';
import { CreateNewUser } from '~/services/api.users';

export const Route = createFileRoute('/signup')({
	component: RouteComponent,
})

function RouteComponent() {
	const [email, setEmail] = useState('');
	const [password, setPassword] = useState('');
	const [confirmPassword, setConfirmPassword] = useState('');
	const [error, setError] = useState('');
	const [isLoading, setIsLoading] = useState(false);
	const navigate = useNavigate()

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		setError('');
		setIsLoading(true);

		// Basic validation
		if (password !== confirmPassword) {
			setError('Passwords do not match');
			setIsLoading(false);
			return;
		}

		if (password.length < 6) {
			setError('Password must be at least 6 characters long');
			setIsLoading(false);
			return;
		}

		try {
			const result = await CreateNewUser({
				email,
				password
			});

			if (result.success) {
				// Redirect to home page or game
				navigate({
					to: '/login'
				})
			} else {
				setError('Failed to create account. Please try again.');
			}
		} catch (err) {
			setError('An error occurred. Please try again.');
		} finally {
			setIsLoading(false);
		}
	};

	return (
		<Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '50vh' }}>
			<Paper sx={{ p: 4, maxWidth: 400, width: '100%' }}>
				<Typography variant="h4" component="h1" gutterBottom align="center">
					Sign Up
				</Typography>

				{error && (
					<Alert severity="error" sx={{ mb: 2 }}>
						{error}
					</Alert>
				)}

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
						autoComplete="new-password"
					/>
					<TextField
						fullWidth
						placeholder="Confirm Password"
						type="password"
						value={confirmPassword}
						onChange={(e) => setConfirmPassword(e.target.value)}
						margin="normal"
						required
						autoComplete="new-password"
					/>
					<Button
						type="submit"
						fullWidth
						variant="contained"
						sx={{ mt: 3, mb: 2 }}
						disabled={isLoading}
					>
						{isLoading ? 'Creating Account...' : 'Sign Up'}
					</Button>
				</form>

				<Typography variant="body2" align="center">
					Already have an account?{' '}
					<Link to="/login" style={{ color: '#1976d2', textDecoration: 'none' }}>
						Sign In
					</Link>
				</Typography>
			</Paper>
		</Box>
	)
}

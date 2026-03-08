# ── Outputs ──────────────────────────────────────────────────────────────────
output "http_games_invoke_arn" { value = aws_lambda_function.http_games.invoke_arn }
output "http_users_invoke_arn" { value = aws_lambda_function.http_users.invoke_arn }
output "http_admin_invoke_arn" { value = aws_lambda_function.http_admin.invoke_arn }
output "http_invites_invoke_arn" { value = aws_lambda_function.http_invites.invoke_arn }
output "ws_connect_invoke_arn" { value = aws_lambda_function.ws_connect.invoke_arn }
output "ws_disconnect_invoke_arn" { value = aws_lambda_function.ws_disconnect.invoke_arn }
output "ws_chat_invoke_arn" { value = aws_lambda_function.ws_chat.invoke_arn }
output "ws_game_action_invoke_arn" { value = aws_lambda_function.ws_game_action.invoke_arn }
output "world_gen_invoke_arn" { value = aws_lambda_function.world_gen.invoke_arn }
output "cognito_post_confirm_invoke_arn" { value = aws_lambda_function.cognito_post_confirm.invoke_arn }
output "cognito_post_confirm_function_arn" { value = aws_lambda_function.cognito_post_confirm.arn }

output "http_games_function_name" { value = aws_lambda_function.http_games.function_name }
output "http_users_function_name" { value = aws_lambda_function.http_users.function_name }
output "http_admin_function_name" { value = aws_lambda_function.http_admin.function_name }
output "http_invites_function_name" { value = aws_lambda_function.http_invites.function_name }
output "ws_connect_function_name" { value = aws_lambda_function.ws_connect.function_name }
output "ws_disconnect_function_name" { value = aws_lambda_function.ws_disconnect.function_name }
output "ws_chat_function_name" { value = aws_lambda_function.ws_chat.function_name }
output "ws_game_action_function_name" { value = aws_lambda_function.ws_game_action.function_name }
output "world_gen_function_name" { value = aws_lambda_function.world_gen.function_name }
output "cognito_post_confirm_function_name" { value = aws_lambda_function.cognito_post_confirm.function_name }

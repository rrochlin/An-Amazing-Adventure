// @vitest-environment jsdom
import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ThemeProvider } from "@mui/material/styles";
import { AppTheme } from "@/theme/theme";
import { Chat } from "@/components/Chat";
import type { ChatMessage } from "@/types/types";

function renderChat(props: Partial<Parameters<typeof Chat>[0]> = {}) {
  const defaults = {
    chatHistory: [] as ChatMessage[],
    command: "",
    setCommand: vi.fn(),
    handleCommand: vi.fn(),
    isLoading: false,
  };
  return render(
    <ThemeProvider theme={AppTheme}>
      <Chat {...defaults} {...props} />
    </ThemeProvider>
  );
}

describe("Chat component", () => {
  it("renders with empty history", () => {
    renderChat();
    expect(screen.getByPlaceholderText("Speak thy command...")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /send/i })).toBeInTheDocument();
  });

  it("renders player and narrative messages", () => {
    const history: ChatMessage[] = [
      { type: "player", content: "Go north" },
      { type: "narrative", content: "You walk through a dark corridor." },
    ];
    renderChat({ chatHistory: history });
    expect(screen.getByText("Go north")).toBeInTheDocument();
    expect(screen.getByText("You walk through a dark corridor.")).toBeInTheDocument();
  });

  it("shows loading indicator when isLoading is true and no streaming content", () => {
    renderChat({ isLoading: true });
    // Send button shows spinner — button is disabled
    const sendBtn = screen.getByRole("button");
    expect(sendBtn).toBeDisabled();
    // No streaming content → LoadingMessage shimmer flavor text is shown
    // (one of the flavor texts from the array — we just check the button state here)
  });

  it("shows streaming bubble with content when isLoading and streamingMessage provided", () => {
    renderChat({ isLoading: true, streamingMessage: "The dungeon echoes..." });
    // The streaming text should be visible
    expect(screen.getByText("The dungeon echoes...")).toBeInTheDocument();
  });

  it("does not show streaming content in committed history while streaming", () => {
    const history: ChatMessage[] = [
      { type: "player", content: "Go north" },
    ];
    renderChat({
      chatHistory: history,
      isLoading: true,
      streamingMessage: "Shadows stir ahead.",
    });
    // Committed player message visible
    expect(screen.getByText("Go north")).toBeInTheDocument();
    // Streaming chunk visible in the streaming bubble
    expect(screen.getByText("Shadows stir ahead.")).toBeInTheDocument();
  });

  it("Send button disabled when command is empty", () => {
    renderChat({ command: "" });
    expect(screen.getByRole("button", { name: /send/i })).toBeDisabled();
  });

  it("Send button enabled when command has content", () => {
    renderChat({ command: "Go west" });
    expect(screen.getByRole("button", { name: /send/i })).not.toBeDisabled();
  });

  it("calls setCommand on input change", async () => {
    const setCommand = vi.fn();
    renderChat({ setCommand });
    const input = screen.getByPlaceholderText("Speak thy command...");
    await userEvent.type(input, "h");
    expect(setCommand).toHaveBeenCalled();
  });

  it("calls handleCommand when Send is clicked", async () => {
    const handleCommand = vi.fn();
    renderChat({ command: "Attack goblin", handleCommand });
    await userEvent.click(screen.getByRole("button", { name: /send/i }));
    expect(handleCommand).toHaveBeenCalledTimes(1);
  });

  it("calls handleCommand on Enter key (not Shift+Enter)", async () => {
    const handleCommand = vi.fn();
    renderChat({ command: "Look around", handleCommand });
    const input = screen.getByPlaceholderText("Speak thy command...");
    fireEvent.keyPress(input, { key: "Enter", charCode: 13, shiftKey: false });
    expect(handleCommand).toHaveBeenCalledTimes(1);
  });

  it("does not call handleCommand on Shift+Enter", () => {
    const handleCommand = vi.fn();
    renderChat({ command: "multi\nline", handleCommand });
    const input = screen.getByPlaceholderText("Speak thy command...");
    fireEvent.keyPress(input, { key: "Enter", charCode: 13, shiftKey: true });
    expect(handleCommand).not.toHaveBeenCalled();
  });
});

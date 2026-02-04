import { render, screen } from "@testing-library/react";
import { ThemeToggle } from "../ThemeToggle";

test("renders theme toggle button", () => {
  render(<ThemeToggle />);

  const btn = screen.getByRole("button", { name: /toggle theme/i });
  expect(btn).toBeInTheDocument();
});

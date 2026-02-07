import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { Header } from "../Header";

describe("Header Component", () => {

  const mockProps = {
    health: null,
    status: "connected" as any,
    error: null,
    onSettingsClick: jest.fn(),
    onReconnect: jest.fn()
  };

  test("renders without crashing", () => {
    render(
      <MemoryRouter>
        <Header {...mockProps} />
      </MemoryRouter>
    );
  });

  test("shows header title", () => {
    render(
      <MemoryRouter>
        <Header {...mockProps} />
      </MemoryRouter>
    );

    const heading = screen.getByRole("heading", { name: /log/i });
    expect(heading).toBeInTheDocument();
  });

});

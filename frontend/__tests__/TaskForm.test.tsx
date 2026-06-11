import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import TaskForm from "@/components/TaskForm";

describe("TaskForm", () => {
  it("shows a validation error and does not submit when title is empty", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn().mockResolvedValue(undefined);

    render(<TaskForm submitLabel="Create task" onSubmit={onSubmit} />);

    await user.click(screen.getByRole("button", { name: "Create task" }));

    expect(await screen.findByText("Title is required.")).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("submits the form with entered values", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn().mockResolvedValue(undefined);

    render(<TaskForm submitLabel="Create task" onSubmit={onSubmit} />);

    await user.type(screen.getByLabelText(/title/i), "Write tests");
    await user.type(screen.getByLabelText("Description"), "Cover the form");
    await user.selectOptions(screen.getByLabelText("Priority"), "high");
    await user.click(screen.getByRole("button", { name: "Create task" }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({
        title: "Write tests",
        description: "Cover the form",
        priority: "high",
        status: "todo",
      })
    );
  });

  it("renders a delete button only when onDelete is provided", () => {
    const onSubmit = vi.fn();
    const { rerender } = render(<TaskForm submitLabel="Save" onSubmit={onSubmit} />);
    expect(screen.queryByRole("button", { name: "Delete task" })).not.toBeInTheDocument();

    rerender(<TaskForm submitLabel="Save" onSubmit={onSubmit} onDelete={vi.fn()} />);
    expect(screen.getByRole("button", { name: "Delete task" })).toBeInTheDocument();
  });
});

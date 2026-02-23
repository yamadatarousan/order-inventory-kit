import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { OrderCreateForm } from "./OrderCreateForm";
import type { CreateOrderResult } from "./orderCreateApi";

type SubmitOrder = (
  input: {
    customerId: string;
    sku: string;
    quantity: number;
    unitPrice: number;
  },
) => Promise<CreateOrderResult>;

describe("OrderCreateForm", () => {
  it("入力欄と送信ボタンを表示する", () => {
    render(<OrderCreateForm />);

    expect(screen.getByLabelText("customerId")).toBeInTheDocument();
    expect(screen.getByLabelText("sku")).toBeInTheDocument();
    expect(screen.getByLabelText("quantity")).toBeInTheDocument();
    expect(screen.getByLabelText("unitPrice")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "注文を作成" }),
    ).toBeInTheDocument();
  });

  it("入力して送信すると注文作成が実行される", async () => {
    const user = userEvent.setup();
    const submitOrder = vi.fn().mockResolvedValue({
      kind: "success",
      orderId: "order-1",
      status: "accepted",
    });

    render(<OrderCreateForm submitOrder={submitOrder as SubmitOrder} />);

    await user.type(screen.getByLabelText("customerId"), "c-1");
    await user.type(screen.getByLabelText("sku"), "sku-1");
    await user.type(screen.getByLabelText("quantity"), "1");
    await user.type(screen.getByLabelText("unitPrice"), "100");
    await user.click(screen.getByRole("button", { name: "注文を作成" }));

    expect(submitOrder).toHaveBeenCalledWith({
      customerId: "c-1",
      sku: "sku-1",
      quantity: 1,
      unitPrice: 100,
    });
  });

  it("400のとき入力エラーを表示する", async () => {
    const user = userEvent.setup();
    const submitOrder = vi.fn().mockResolvedValue({
      kind: "invalid",
    });

    render(<OrderCreateForm submitOrder={submitOrder as SubmitOrder} />);

    await user.type(screen.getByLabelText("customerId"), "c-1");
    await user.type(screen.getByLabelText("sku"), "sku-1");
    await user.type(screen.getByLabelText("quantity"), "1");
    await user.type(screen.getByLabelText("unitPrice"), "100");
    await user.click(screen.getByRole("button", { name: "注文を作成" }));

    expect(await screen.findByText("入力内容を確認してください")).toBeInTheDocument();
  });

  it("409のとき価格不一致エラーを表示する", async () => {
    const user = userEvent.setup();
    const submitOrder = vi.fn().mockResolvedValue({
      kind: "priceConflict",
    });

    render(<OrderCreateForm submitOrder={submitOrder as SubmitOrder} />);

    await user.type(screen.getByLabelText("customerId"), "c-1");
    await user.type(screen.getByLabelText("sku"), "sku-1");
    await user.type(screen.getByLabelText("quantity"), "1");
    await user.type(screen.getByLabelText("unitPrice"), "100");
    await user.click(screen.getByRole("button", { name: "注文を作成" }));

    expect(
      await screen.findByText("価格が更新されました。再確認してください"),
    ).toBeInTheDocument();
  });
});

import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { OrderActionsPanel } from "./OrderActionsPanel";
import type {
  CancelOrderResult,
  ConfirmPaymentInput,
  ConfirmPaymentResult,
  FetchOrderResult,
} from "./orderActionsApi";

type FetchOrder = (orderId: string) => Promise<FetchOrderResult>;
type CancelOrder = (orderId: string) => Promise<CancelOrderResult>;
type ConfirmPayment = (
  input: ConfirmPaymentInput,
) => Promise<ConfirmPaymentResult>;

describe("OrderActionsPanel", () => {
  it("現在の注文IDで注文参照し主要項目を表示する", async () => {
    const user = userEvent.setup();
    const fetchOrder = vi.fn().mockResolvedValue({
      kind: "success",
      order: {
        id: "order-1",
        customerId: "c-1",
        status: "accepted",
        items: [{ sku: "sku-1", quantity: 2 }],
      },
    });

    render(
      <OrderActionsPanel
        fetchOrder={fetchOrder as FetchOrder}
        currentOrderId="order-1"
      />,
    );

    await user.click(screen.getByRole("button", { name: "注文を再取得" }));

    expect(fetchOrder).toHaveBeenCalledWith("order-1");
    expect(await screen.findByText("id: order-1")).toBeInTheDocument();
    expect(screen.getByText("customerId: c-1")).toBeInTheDocument();
    expect(screen.getByText("status: accepted")).toBeInTheDocument();
    expect(screen.getByText("sku-1 x 2")).toBeInTheDocument();
  });

  it("注文参照404のとき not found を表示する", async () => {
    const user = userEvent.setup();
    const fetchOrder = vi.fn().mockResolvedValue({
      kind: "notFound",
    });

    render(
      <OrderActionsPanel
        fetchOrder={fetchOrder as FetchOrder}
        currentOrderId="missing"
      />,
    );

    await user.click(screen.getByRole("button", { name: "注文を再取得" }));

    expect(await screen.findByText("注文が見つかりません")).toBeInTheDocument();
  });

  it("注文キャンセルを実行できる", async () => {
    const user = userEvent.setup();
    const cancelOrder = vi.fn().mockResolvedValue({
      kind: "success",
      order: {
        id: "order-1",
        customerId: "c-1",
        status: "canceled",
        items: [{ sku: "sku-1", quantity: 1 }],
      },
    });

    render(
      <OrderActionsPanel
        cancelOrder={cancelOrder as CancelOrder}
        currentOrderId="order-1"
      />,
    );

    await user.click(screen.getByRole("button", { name: "注文をキャンセル" }));

    expect(cancelOrder).toHaveBeenCalledWith("order-1");
    expect(await screen.findByText("注文をキャンセルしました")).toBeInTheDocument();
    expect(screen.getByText("status: canceled")).toBeInTheDocument();
  });

  it("決済確定で200を表示する", async () => {
    const user = userEvent.setup();
    const confirmPayment = vi.fn().mockResolvedValue({
      kind: "success",
      orderId: "order-1",
      paymentStatus: "confirmed",
    });

    render(
      <OrderActionsPanel
        confirmPayment={confirmPayment as ConfirmPayment}
        currentOrderId="order-1"
      />,
    );

    await user.type(screen.getByLabelText("amount"), "100");
    await user.type(screen.getByLabelText("idempotencyKey"), "k-1");
    await user.click(screen.getByRole("button", { name: "決済を確定" }));

    expect(confirmPayment).toHaveBeenCalledWith({
      orderId: "order-1",
      amount: 100,
      idempotencyKey: "k-1",
    });
    expect(await screen.findByText("決済を確定しました")).toBeInTheDocument();
  });

  it("決済確定で409を表示する", async () => {
    const user = userEvent.setup();
    const confirmPayment = vi.fn().mockResolvedValue({
      kind: "amountConflict",
    });

    render(
      <OrderActionsPanel
        confirmPayment={confirmPayment as ConfirmPayment}
        currentOrderId="order-1"
      />,
    );

    await user.type(screen.getByLabelText("amount"), "100");
    await user.type(screen.getByLabelText("idempotencyKey"), "k-1");
    await user.click(screen.getByRole("button", { name: "決済を確定" }));

    expect(await screen.findByText("金額が一致しません")).toBeInTheDocument();
  });

  it("現在の注文IDを表示する", () => {
    render(<OrderActionsPanel currentOrderId="order-xyz" />);

    expect(screen.getByText("現在の注文ID: order-xyz")).toBeInTheDocument();
  });

  it("現在の注文IDがない場合は先に作成を促す", async () => {
    const user = userEvent.setup();
    render(<OrderActionsPanel />);

    await user.click(screen.getByRole("button", { name: "注文を再取得" }));
    expect(
      await screen.findByText("先に注文を作成してください"),
    ).toBeInTheDocument();
  });
});

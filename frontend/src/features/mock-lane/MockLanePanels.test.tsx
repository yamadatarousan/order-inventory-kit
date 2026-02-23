import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MockLanePanels } from "./MockLanePanels";

describe("MockLanePanels", () => {
  it("販売中商品のみを一覧表示する", async () => {
    render(<MockLanePanels />);

    expect(await screen.findByText(/sku-1/)).toBeInTheDocument();
    expect(screen.getByText(/sku-2/)).toBeInTheDocument();
    expect(screen.queryByText(/sku-stop/)).not.toBeInTheDocument();
  });

  it("商品詳細で存在しないSKUは404メッセージを表示する", async () => {
    const user = userEvent.setup();
    render(<MockLanePanels />);

    await user.type(screen.getByLabelText("商品詳細SKU"), "missing-sku");
    await user.click(screen.getByRole("button", { name: "商品詳細を表示" }));

    expect(await screen.findByText("商品が見つかりません (404)")).toBeInTheDocument();
  });

  it("カート追加で無効SKU/無効数量は400メッセージを表示する", async () => {
    const user = userEvent.setup();
    render(<MockLanePanels />);

    await user.type(screen.getByLabelText("カートSKU"), "missing-sku");
    await user.type(screen.getByLabelText("カート数量"), "1");
    await user.click(screen.getByRole("button", { name: "カートに追加" }));
    expect(await screen.findByText("カート追加に失敗しました (400)")).toBeInTheDocument();
  });

  it("カート内容で明細とサーバ算出合計を表示する", async () => {
    const user = userEvent.setup();
    render(<MockLanePanels />);

    await user.type(screen.getByLabelText("カートSKU"), "sku-1");
    await user.type(screen.getByLabelText("カート数量"), "2");
    await user.click(screen.getByRole("button", { name: "カートに追加" }));

    expect(await screen.findByText("sku-1 x 2 = 200")).toBeInTheDocument();
    expect(screen.getByText("subtotal: 200")).toBeInTheDocument();
    expect(screen.getByText("shipping: 500")).toBeInTheDocument();
    expect(screen.getByText("serviceFee: 100")).toBeInTheDocument();
    expect(screen.getByText("total: 800 (server-calculated mock)")).toBeInTheDocument();
  });

  it("数量更新で反映され、無効数量は400メッセージを表示する", async () => {
    const user = userEvent.setup();
    render(<MockLanePanels />);

    await user.type(screen.getByLabelText("カートSKU"), "sku-1");
    await user.type(screen.getByLabelText("カート数量"), "1");
    await user.click(screen.getByRole("button", { name: "カートに追加" }));

    await user.type(screen.getByLabelText("更新数量"), "3");
    await user.click(screen.getByRole("button", { name: "数量更新" }));
    expect(await screen.findByText("sku-1 x 3 = 300")).toBeInTheDocument();

    await user.clear(screen.getByLabelText("更新数量"));
    await user.type(screen.getByLabelText("更新数量"), "0");
    await user.click(screen.getByRole("button", { name: "数量更新" }));
    expect(await screen.findByText("数量変更に失敗しました (400)")).toBeInTheDocument();
  });

  it("最終確認で小計/送料/手数料/合計を表示する", async () => {
    const user = userEvent.setup();
    render(<MockLanePanels />);

    await user.type(screen.getByLabelText("カートSKU"), "sku-2");
    await user.type(screen.getByLabelText("カート数量"), "1");
    await user.click(screen.getByRole("button", { name: "カートに追加" }));

    const section = screen.getByText("最終確認（US-CHK-03）").closest("section");
    expect(section).not.toBeNull();
    const scoped = within(section!);
    expect(scoped.getByText("商品小計: 250")).toBeInTheDocument();
    expect(scoped.getByText("送料: 500")).toBeInTheDocument();
    expect(scoped.getByText("手数料: 100")).toBeInTheDocument();
    expect(scoped.getByText("合計: 850")).toBeInTheDocument();
    expect(scoped.getByText("表示基準: サーバ算出値（mock）")).toBeInTheDocument();
  });

  it("履歴一覧と履歴詳細を表示する", async () => {
    const user = userEvent.setup();
    render(<MockLanePanels />);

    await user.clear(screen.getByLabelText("履歴customerId"));
    await user.type(screen.getByLabelText("履歴customerId"), "c-1");
    await user.click(screen.getByRole("button", { name: "履歴を取得" }));

    expect(await screen.findByText(/hist-1 \/ confirmed/)).toBeInTheDocument();
    expect(screen.getByText(/hist-2 \/ accepted/)).toBeInTheDocument();

    await user.click(screen.getAllByRole("button", { name: "詳細を表示" })[0]);
    expect(await screen.findByText("detail-orderId: hist-1")).toBeInTheDocument();
    expect(screen.getByText("detail-total: 1200")).toBeInTheDocument();
  });
});

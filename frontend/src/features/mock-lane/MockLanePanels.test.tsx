import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MockLanePanels } from "./MockLanePanels";

describe("MockLanePanels", () => {
  it("ログイン後に販売中商品のみを一覧表示する", async () => {
    const user = userEvent.setup();
    render(<MockLanePanels />);

    await user.clear(screen.getByLabelText("会員ID"));
    await user.type(screen.getByLabelText("会員ID"), "c-1");
    await user.type(screen.getByLabelText("パスワード"), "pw");
    await user.click(screen.getByRole("button", { name: "ログイン" }));

    expect(await screen.findByText(/sku-1/)).toBeInTheDocument();
    expect(screen.getByText(/sku-2/)).toBeInTheDocument();
    expect(screen.queryByText(/sku-stop/)).not.toBeInTheDocument();
  });

  it("検索で商品を絞り込める", async () => {
    const user = userEvent.setup();
    render(<MockLanePanels />);

    await user.clear(screen.getByLabelText("会員ID"));
    await user.type(screen.getByLabelText("会員ID"), "c-1");
    await user.type(screen.getByLabelText("パスワード"), "pw");
    await user.click(screen.getByRole("button", { name: "ログイン" }));

    await user.type(screen.getByLabelText("検索"), "sku-2");

    expect(await screen.findByText(/sku-2/)).toBeInTheDocument();
    expect(screen.queryByText(/sku-1/)).not.toBeInTheDocument();
  });

  it("商品をカート追加して最終確認の合計を表示する", async () => {
    const user = userEvent.setup();
    render(<MockLanePanels />);

    await user.clear(screen.getByLabelText("会員ID"));
    await user.type(screen.getByLabelText("会員ID"), "c-1");
    await user.type(screen.getByLabelText("パスワード"), "pw");
    await user.click(screen.getByRole("button", { name: "ログイン" }));

    await user.clear(screen.getByLabelText("数量", { selector: "#product-qty-sku-1" }));
    await user.type(screen.getByLabelText("数量", { selector: "#product-qty-sku-1" }), "2");
    await user.click(screen.getAllByRole("button", { name: "カートに追加" })[0]);

    expect(await screen.findByText("sku-1 x 2 = 200")).toBeInTheDocument();
    expect(screen.getByText("商品小計: 200")).toBeInTheDocument();
    expect(screen.getByText("送料: 500")).toBeInTheDocument();
    expect(screen.getByText("手数料: 100")).toBeInTheDocument();
    expect(screen.getByText("合計: 800")).toBeInTheDocument();
  });

  it("数量更新で反映される", async () => {
    const user = userEvent.setup();
    render(<MockLanePanels />);

    await user.clear(screen.getByLabelText("会員ID"));
    await user.type(screen.getByLabelText("会員ID"), "c-1");
    await user.type(screen.getByLabelText("パスワード"), "pw");
    await user.click(screen.getByRole("button", { name: "ログイン" }));

    await user.click(screen.getAllByRole("button", { name: "カートに追加" })[0]);
    await user.clear(screen.getByLabelText("数量変更"));
    await user.type(screen.getByLabelText("数量変更"), "3");
    await user.click(screen.getByRole("button", { name: "数量更新" }));
    expect(await screen.findByText("sku-1 x 3 = 300")).toBeInTheDocument();
  });

  it("履歴一覧と履歴詳細を表示する", async () => {
    const user = userEvent.setup();
    render(<MockLanePanels />);

    await user.clear(screen.getByLabelText("会員ID"));
    await user.type(screen.getByLabelText("会員ID"), "c-1");
    await user.type(screen.getByLabelText("パスワード"), "pw");
    await user.click(screen.getByRole("button", { name: "ログイン" }));

    await user.click(screen.getByRole("button", { name: "履歴を取得" }));

    expect(await screen.findByText(/hist-1 \/ confirmed/)).toBeInTheDocument();
    expect(screen.getByText(/hist-2 \/ accepted/)).toBeInTheDocument();

    await user.click(screen.getAllByRole("button", { name: "詳細を表示" })[0]);
    expect(await screen.findByText("detail-orderId: hist-1")).toBeInTheDocument();
    expect(screen.getByText("detail-total: 1200")).toBeInTheDocument();
  });
});

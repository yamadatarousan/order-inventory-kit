import { useState } from "react";
import { OrderCreateForm } from "../features/order-create/OrderCreateForm";
import { OrderActionsPanel } from "../features/order-actions/OrderActionsPanel";
import { MockLanePanels } from "../features/mock-lane/MockLanePanels";

export function App() {
  const [latestOrderId, setLatestOrderId] = useState("");

  return (
    <main className="app-shell">
      <header className="app-hero">
        <p className="app-eyebrow">ORDER INVENTORY KIT</p>
        <h1>ミニECフロント（仮実装 + API接続）</h1>
        <p className="app-lead">
          EC導線を先に確認できる仮実装画面を主表示し、下段にAPI接続検証パネルを配置しています。
        </p>
      </header>
      <div className="app-grid">
        <MockLanePanels />
      </div>
      <section className="panel dev-panel">
        <h2>API接続検証（開発用）</h2>
        <p>本番導線とは分離した、注文APIの疎通確認パネルです。</p>
        <div className="app-grid app-grid-dev">
          <OrderCreateForm onCreated={setLatestOrderId} />
          <OrderActionsPanel currentOrderId={latestOrderId} />
        </div>
      </section>
    </main>
  );
}

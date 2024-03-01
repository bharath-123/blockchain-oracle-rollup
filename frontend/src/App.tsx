import { useEffect, useRef, useState } from "react";
import "./App.css";
import "./global.css";

// Define a type for the message object
type EthData = {
  blockHash: string;
  stateRoot: string;
  parentRoot: string;
  slot: number;
  propserIndex: number;
};

// bestest random ever
const senderId = Math.random().toString(36).substr(2);

function App() {
  // Use the Message type for the messages state
  const [ethDatas, setEthDatas] = useState<EthData[]>([]);
  const ws = useRef<WebSocket | null>(null);

  // Set latest message ref
  const endOfMessagesRef = useRef<null | HTMLDivElement>(null);

  const sender = `user-${senderId}`;

  // get messages from rollup ws
  useEffect(() => {
    ws.current = new WebSocket("ws://localhost:8080/ws");
    ws.current.onmessage = (event) => {
      console.log(`event.data is `);
      console.log(event.data);
      const data = JSON.parse(event.data);
      const message: EthData = {
        blockHash: data.block_hash,
        stateRoot: data.state_root,
        parentRoot: data.parent_root,
        slot: data.slot,
        propserIndex: data.proposer_index,
      };
      setEthDatas((prevMessages) => [...prevMessages, message]);
    };
    return () => {
      ws.current?.close();
    };
  }, [sender]);

  // Snap to latest message
  useEffect(() => {
    endOfMessagesRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [ethDatas]);

  return (
    <>
      <div className="nes-container with-title is-centered">
        <div className="title">
          <p>Modular Chat</p>
        </div>
        <div className="message-list">
          {ethDatas.map((ethData, index) => (
            <section key={index}>
              <p>Block Hash: {ethData.blockHash}</p>
              <p>Parent Hash: {ethData.parentRoot}</p>{" "}
              <p>State Root: {ethData.stateRoot}</p> <p>Slot: {ethData.slot}</p>
              <p>Proposer Index: {ethData.propserIndex}</p>
            </section>
          ))}
          <div ref={endOfMessagesRef} />
        </div>
      </div>
      <div className="footer">
        <p>
          built by{" "}
          <a href="https://astria.org" target="_blank">
            Astria
          </a>{" "}
          with{" "}
          <a href="https://celestia.org/" target="_blank">
            Celestia
          </a>{" "}
          underneath
        </p>
        <p>
          <a href="https://twitter.com/AstriaOrg" target="_blank">
            <i className="nes-icon close is-medium"></i>
          </a>
          <a
            href="https://github.com/astriaorg/messenger-rollup"
            target="_blank"
          >
            <i className="nes-icon github is-medium"></i>
          </a>
        </p>
      </div>
    </>
  );
}

export default App;

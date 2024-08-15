import ws from "k6/ws";
import { check, fail, sleep } from "k6";

export const options = {
  scenarios: {
    "constant-vus": {
      executor: "constant-vus",
      vus: 100,
      duration: "1m",
    },
    "shared-iterations": {
      executor: "shared-iterations",
      vus: 100,
      iterations: 1000,
      maxDuration: "5m",
    },
  },
};

const MESSAGES_PER_VU = 5;
const SLEEP_BETWEEN_MESSAGES = 1;
const CHECKBOXES = 1000000000;

export default function () {
  const res = ws.connect(__ENV.WEBSOCKET_URL, function (socket) {
    socket.on("open", function open() {
      console.log(`VU ${__VU}: connected`);

      socket.setTimeout(() => {
        for (let i = 0; i < MESSAGES_PER_VU; i++) {
          const value = Math.floor(Math.random() * CHECKBOXES) + 1;

          const message = new ArrayBuffer(4);
          const view = new DataView(message);
          view.setUint32(0, value, true);

          socket.sendBinary(message);

          console.log(`VU ${__VU}: value sent:`, value);

          sleep(Math.random() * SLEEP_BETWEEN_MESSAGES);
        }

        socket.close();
      }, 1);

      socket.setInterval(function timeout() {
        socket.ping();
        console.log("Ping sent!");
      }, 60 * 1000);
    });

    socket.on("ping", function () {
      console.log("Ping received!");
    });

    socket.on("pong", function () {
      console.log("Pong received!");
    });

    socket.on("close", function () {
      console.log(`VU ${__VU}: close`);
    });

    socket.on("error", function (e) {
      console.error(`VU ${__VU}: error: ${e.error()}`);
      check(false, { "Has error": (val) => val });
    });

    socket.on("binaryMessage", function (message) {
      check(message, {
        "Message byte length is 4": (message) => message.byteLength === 8 || message.byteLength === 4,
      });
    });

    socket.setTimeout(function () {
      socket.close();
    }, SLEEP_BETWEEN_MESSAGES * MESSAGES_PER_VU * 1000 + 1000);
  });

  check(res, { "Connected successfully": (r) => r && r.status === 101 });
}

import http from "k6/http";
import { check } from "k6";

const BASE_URL = "https://whoami.kapsule.camier.family";

const stages = [
  { duration: "30s", target: 100 },
  { duration: "3m", target: 100 },
  { duration: "1m", target: 500 },
  { duration: "5m", target: 500 },
  { duration: "1m", target: 1000 },
  { duration: "5m", target: 1000 },
  { duration: "2m", target: 1500 },
  { duration: "2m", target: 1500 },
  { duration: "30s", target: 0 },
];

export const options = {
  stages,
  thresholds: {
    http_req_duration: ["p(95)<2000"],
  },
};

export default function () {
  const normal = http.get(BASE_URL + "/");
  check(normal, { "normal 200": (r) => r.status === 200 });

  const xss = http.get(
    BASE_URL + "/?test=%3Cscript%3Ealert(1)%3C%2Fscript%3E"
  );
  check(xss, { "xss blocked": (r) => r.status === 403 });
}

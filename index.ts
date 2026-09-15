import { loadConfig } from "./src/config.ts";
import { runCli } from "./src/cli.ts";
import { runTui } from "./src/tui/app.ts";

const noUi = Bun.argv.includes("--noui");
const config = loadConfig();

if (noUi) {
  await runCli(config);
} else {
  await runTui(config);
}

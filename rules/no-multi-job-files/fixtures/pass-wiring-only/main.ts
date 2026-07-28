import { parseArgs } from "./args";
import { collectFindings } from "./collect";
import { writeReport } from "./report";

async function main(argv: string[]): Promise<number> {
  const options = parseArgs(argv);
  if (options.help) {
    process.stdout.write(usage);
    return 0;
  }

  const findings = await collectFindings(options.paths, options.offline);
  writeReport(process.stdout, options.format, findings);
  return findings.length > 0 ? 1 : 0;
}

const usage = `scan [--format text|json] [--offline] [path...]\n`;

main(process.argv.slice(2)).then((code) => process.exit(code));

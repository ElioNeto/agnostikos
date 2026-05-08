import { execSync } from "child_process";

const files = [
  "internal/bootstrap/kernel_test.go",
  "internal/bootstrap/iso_test.go",
  "scripts/test-boot-integration.sh",
];

for (const f of files) {
  console.log(`Staging ${f}...`);
  execSync(`git add ${f}`, { stdio: "inherit" });
}

console.log("Done.");

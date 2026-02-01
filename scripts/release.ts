import { $ } from "bun";

async function main() {
  // Check we're on master branch
  const currentBranch = (await $`git branch --show-current`.text()).trim();
  if (currentBranch !== "master") {
    console.error(`Error: Must be on master branch (currently on ${currentBranch})`);
    process.exit(1);
  }

  // Fetch latest and check if up to date
  await $`git fetch origin master`;
  const localCommit = (await $`git rev-parse HEAD`.text()).trim();
  const remoteCommit = (await $`git rev-parse origin/master`.text()).trim();

  if (localCommit !== remoteCommit) {
    console.error("Error: Local master is not up to date with origin/master");
    console.error(`  Local:  ${localCommit}`);
    console.error(`  Remote: ${remoteCommit}`);
    process.exit(1);
  }

  // Generate date-based tag (YYYY-MM-DD)
  const today = new Date();
  const baseTag = today.toISOString().split("T")[0];
  let tag = baseTag;
  let suffix = 0;

  // Find unique tag
  const existingTags = (await $`git tag -l`.text()).split("\n");
  while (existingTags.includes(tag)) {
    suffix++;
    tag = `${baseTag}-${suffix}`;
  }

  console.log(`Creating release with tag: ${tag}`);
  await $`gh release create ${tag} --generate-notes`;
}

main();

#!/usr/bin/env node
import { Command, CommanderError } from 'commander'
import { cyan, yellow } from 'picocolors'
import packageJson from './package.json'
import { getProjectInputs } from './inputs'
import { createApp } from './helpers/create-app'

const program = new Command(packageJson.name)
  .version(packageJson.version)
  .argument('[directory]')
  .usage('[directory] [options]')
  .helpOption('-h, --help', 'Display this help message.')
  .option('--ts, --typescript', 'Initialize as a TypeScript project. (default)')
  .option('--js, --javascript', 'Initialize as a JavaScript project.')
  .parse(process.argv)

const [directory] = program.args

async function run(): Promise<void> {
  try {
    const inputs = await getProjectInputs(directory)
    await createApp(inputs)
    process.exit(0)
  } catch (reason) {
    if (!(reason instanceof Error)) {
      throw reason
    }

    const error = reason as CommanderError
    if (error) {
      console.error(`Failed to execute ${error}`)
    } else {
      console.error(error)
    }
    process.exit(1)
  }
}

run().catch(async (reason) => {
  console.log()
  console.log('Aborting installation.')
  const error = reason as any
  if (error.command) {
    console.log(`  ${cyan(error.command)} has failed.`)
  } else {
    console.log(
      yellow('Unexpected error. Please report it as a bug:') + '\n',
      reason
    )
  }
  console.log()

  process.exit(1)
}) 
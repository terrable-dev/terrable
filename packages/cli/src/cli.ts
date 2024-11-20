#!/usr/bin/env node

import { Command } from "commander";
import { exec, spawn } from "child_process";
import * as path from "path";

const program = new Command();
const binaryPath = path.resolve(__dirname, "../terrable"); // Adjust the path to your binary

program.name("terrable-cli").description("CLI for Terrable").version("1.0.0");

program
  .command("offline")
  .description("Run the offline server")
  .option("-f, --file <file>", "Path to the Terraform file")
  .option("-m, --module <module>", "Name of the terraform module")
  .option("-p, --port <port>", "Port number")
  .action((options) => {
    const args = [
      "offline",
      options.file ? `-file ${options.file}` : "",
      options.module ? `-module ${options.module}` : "",
      options.port ? `-port ${options.port}` : "",
    ]
      .filter(Boolean)
      .join(" ");

    const child = spawn(binaryPath, args.split(" "));

    child.stdout.on("data", (data) => {
      process.stdout.write(data);
    });

    child.stderr.on("data", (data) => {
      process.stderr.write(data);
    });

    child.on("close", (code) => {
      console.log(`Child process exited with code ${code}`);
    });
  });

program.parse(process.argv);

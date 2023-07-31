import chalk from "chalk";

class Logger {
  static info(s: string) {
    console.log(s);
  }

  // eslint-disable-next-line
  static deepInfo(s: any) {
    console.log(s);
  }

  static title(s: string) {
    console.log(chalk.blue(s));
  }

  static warn(s: string) {
    console.log(chalk.yellow(s));
  }

  static success(s: string) {
    console.log(chalk.green(s));
  }

  static error(s: string) {
    console.log(chalk.red(s));
  }

  static emptyLine() {
    console.log();
  }
}

export default Logger;

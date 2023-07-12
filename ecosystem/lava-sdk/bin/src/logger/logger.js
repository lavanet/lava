"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const chalk_1 = __importDefault(require("chalk"));
class Logger {
    static info(s) {
        console.log(s);
    }
    // eslint-disable-next-line
    static deepInfo(s) {
        console.log(s);
    }
    static title(s) {
        console.log(chalk_1.default.blue(s));
    }
    static warn(s) {
        console.log(chalk_1.default.yellow(s));
    }
    static success(s) {
        console.log(chalk_1.default.green(s));
    }
    static error(s) {
        console.log(chalk_1.default.red(s));
    }
    static emptyLine() {
        console.log();
    }
}
exports.default = Logger;

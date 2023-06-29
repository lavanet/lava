declare class Logger {
    static info(s: string): void;
    static deepInfo(s: any): void;
    static title(s: string): void;
    static warn(s: string): void;
    static success(s: string): void;
    static error(s: string): void;
    static emptyLine(): void;
}
export default Logger;

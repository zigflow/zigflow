export default {
  extends: ["@commitlint/config-conventional"],
  ignores: [
    (message) =>
      message.startsWith("deps:") ||
      message.startsWith("deps(") ||
      message.includes("ci(deps):"),
  ],
};

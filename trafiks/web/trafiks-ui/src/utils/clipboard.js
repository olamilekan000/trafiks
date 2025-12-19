import ClipboardJS from "clipboard";

/**
 * Copy text to clipboard using clipboardjs
 * @param {string} text - Text to copy
 * @returns {Promise<void>}
 */
export function copyToClipboard(text) {
  return new Promise((resolve, reject) => {
    const tempButton = document.createElement("button");
    tempButton.style.display = "none";
    document.body.appendChild(tempButton);

    const clipboard = new ClipboardJS(tempButton, {
      text: () => text,
    });

    clipboard.on("success", () => {
      clipboard.destroy();
      document.body.removeChild(tempButton);
      resolve();
    });

    clipboard.on("error", (err) => {
      clipboard.destroy();
      document.body.removeChild(tempButton);
      reject(err);
    });

    tempButton.click();
  });
}

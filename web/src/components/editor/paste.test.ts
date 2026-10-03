import { Editor } from "@tiptap/react";
import { expect, it } from "vite-plus/test";
import { createEditorExtensions, getEditorMarkdown } from "./tiptapExtensions";

it("preserves normal HTML paste, undo and redo with the application editor extensions", () => {
  const editor = new Editor({
    element: document.createElement("div"),
    extensions: createEditorExtensions("Write a note"),
    content: "",
  });
  try {
    expect(
      editor.view.pasteHTML(
        "<p>TradeLens <strong>journal</strong></p>",
        new Event("paste") as ClipboardEvent,
      ),
    ).toBe(true);
    expect(getEditorMarkdown(editor)).toBe("TradeLens **journal**");
    expect(editor.commands.undo()).toBe(true);
    expect(editor.getText()).toBe("");
    expect(editor.commands.redo()).toBe(true);
    expect(getEditorMarkdown(editor)).toBe("TradeLens **journal**");
  } finally {
    editor.destroy();
  }
});

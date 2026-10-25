# Style guide: Bogie Depot

How the book is written. Follow this when drafting or editing any chapter material. The model is the Pragmatic Bookshelf style (for example, *Hotwire Native for Rails Developers*): one project, built step by step, by an author who has done it for real. Borrow the shape and the warmth; never copy another book's phrasing.

## Voice

- **The author is a real person with a reason.** Mac built Bogie because he missed Rails' conventions in Go. Let that experience show where it helps the reader trust a decision ("I've shipped apps both ways, and here's what bit me"). Don't overdo it: a sentence, not a story, outside the introduction.
- **"You" do the work; "we" do it together.** Instructions address the reader directly ("Open `app/controllers/products_controller.go`"). Shared discovery uses "we" ("Let's see what Bogie generated").
- **Conversational, short asides.** An occasional reaction ("Much better.", "That's it, really.") keeps the pairing feel. One or two per section at most, and never in place of an explanation.
- **Honest opinions, with reasons.** Say what you prefer and why ("I'd rather offset pages here, because the shop wants 'page 2 of 5'"). Readers should learn judgment, not just steps.
- **Plain English for an international audience.** Short sentences. Explain every term the first time it appears. No idioms that only make sense to native speakers. American spelling.
- **Warm, not salesy.** Enthusiasm comes from the reader's progress ("Your cart now survives a restart"), not from adjectives about Bogie.

## Chapter structure

Every chapter in Part II follows this shape:

1. **Hook.** One or two paragraphs: the client's request, in the shop owner's words, and why it matters to the shop.
2. **What you'll learn.** A short numbered list of the techniques in this chapter.
3. **The early win.** Within the first few pages, the reader runs something and sees it work in the UI or the terminal. Explanation comes after the win, not before.
4. **Sections named after actions.** "Add a Cart to Every Visitor", not "Sessions". The section title tells the reader what they'll have done by the end of it.
5. **What's Next?** A short recap of what the reader built, then the next problem in one or two sentences, as a teaser.

Part I chapters (setup, how the web works, Go essentials) keep the hook and "What's Next?", but may be organized by concept.

## The rhythm inside a section

Each section runs the same loop:

1. **The problem.** Something in the current app is missing, wrong, or awkward. Show it.
2. **A small primer,** only if the concept is new: a tiny standalone example, before touching the app.
3. **Generate or change.** Run the Bogie command or make the edit, with a labeled code listing.
4. **Run it.** Show the result: a UI screenshot, terminal output, or an `.http` request and its JSON.
5. **Read it.** Walk through the important generated or written lines, naming the tool behind each piece (Gin, sqlc, goose).
6. **Reflect.** The trade-off, the limit, or the better approach. If there's a simpler way that has a cost, show the simple way first, name the cost, then show the better way.

Step 6 is what makes this a book and not a tutorial. `docs/build-notes.md` ("Chapter-worthy observations" and "Rails comparisons") is the main source for these moments.

## Code listings

- **Label every listing** with its checkpoint and path, for example `ch06_03/app/controllers/products_controller.go`.
- **Show only what matters.** Collapse unchanged parts with `// ...` in Go, `-- ...` in SQL, and `<!-- ... -->` in HTML.
- **Mark changed lines** with ➤ in the margin.
- **Keep listings short.** If a listing passes about 30 lines, split it or collapse more. Long generated files are walked through in pieces.
- **Every listing must match the checkpoint exactly.** If the code changes, the listing changes.
- **Commands** go in their own block, with output shown separately and trimmed to what the reader needs to see.

## Showing results

- **UI changes:** before-and-after screenshots side by side.
- **API responses:** a request from the chapter's `.http` file and the JSON it returns, trimmed with `...`.
- **Errors on purpose:** when a mistake teaches something, let the reader hit it, show the error message, then fix it.

## Boxes

Use boxes sparingly, at most two or three per chapter.

- **Coming from Rails?** Maps what the reader just did to its Rails equivalent, and says where Go deliberately differs.
- **Under the hood.** For curious readers: what Bogie or a library does internally. The chapter must make sense without it.
- **Working with others.** A collaboration habit tied to what just happened: a changed requirement, a contract question, a code review note.
- **Watch out.** A known problem and its workaround (a version quirk, a common mistake).
- **Background.** A short introduction to a tool or idea some readers won't know.

## Checkpoints

- Every listing and every "if you're stuck" pointer refers to a checkpoint by chapter: `ch06_03` is chapter 6, step 3.
- Each chapter starts from the previous chapter's last checkpoint, and says so in its first section.
- Checkpoints are available as Git tags and as zip downloads.

## Links and references

- URLs go in footnotes, not in the running text.
- Refer to other chapters by number and title: "Chapter 11, Task E: Let Customers Place Orders".
- Name the source when quoting documentation, and keep quotes short.

## Terms and naming

- **Bogie Depot** is the shop. **Bogie** is the tool. Never "the Bogie framework".
- **The client** or **the shop owner** asks for features; **the frontend developer** owns the UI and the contract; **the reviewer** leaves code review notes.
- Prices are always integers in cents (`price_cents`). Say "cents" in prose.
- Use the names the code uses (`price_rules`, `CartItemsController`), formatted as code.

## Before a section is done

- Does it open with a problem the reader can see?
- Does the reader run something and see a result?
- Is every new term explained the first time?
- Does every listing match its checkpoint?
- Is there a reflection: a trade-off, a limit, or a better approach?
- Would a reader who skips every box still follow it?

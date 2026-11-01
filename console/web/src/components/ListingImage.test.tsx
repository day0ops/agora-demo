import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { ListingImage, listingImageSrc } from "./ListingImage";

const baseListing = { image: "", category: "watches", title: "Test Watch" };

describe("listingImageSrc", () => {
  it("uses the real photo path when image is set", () => {
    expect(
      listingImageSrc({ image: "rolex-submariner", category: "watches" }),
    ).toBe("/images/listings/rolex-submariner.jpg");
  });

  it("falls back to the category SVG when image is empty", () => {
    expect(listingImageSrc({ image: "", category: "watches" })).toBe(
      "/images/categories/watches.svg",
    );
  });

  it("normalizes an unknown category to default", () => {
    expect(listingImageSrc({ image: "", category: "snorkel-gear" })).toBe(
      "/images/categories/default.svg",
    );
  });
});

describe("ListingImage", () => {
  it("renders an img with the resolved src and alt text", () => {
    render(<ListingImage listing={baseListing} />);
    const img = screen.getByRole("img", {
      name: "Test Watch",
    }) as HTMLImageElement;
    expect(img.src).toContain("/images/categories/watches.svg");
    // jsdom doesn't implement the `loading` IDL property reflection, so
    // assert on the attribute it does reflect correctly instead.
    expect(img.getAttribute("loading")).toBe("lazy");
  });

  it("swaps to the category fallback on load error", () => {
    render(
      <ListingImage
        listing={{ image: "missing-photo", category: "books", title: "A Book" }}
      />,
    );
    const img = screen.getByRole("img") as HTMLImageElement;
    fireEvent.error(img);
    expect(img.src).toContain("/images/categories/books.svg");
  });
});

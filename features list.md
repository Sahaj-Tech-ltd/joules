# features list

* [ ] fig barcode scanner.

***

## What Fig Actually Does

**Core: The Barcode Scanner** Scan a barcode and instantly see if the product's ingredients are compatible with your dietary needs. You can even tap on a flagged ingredient to learn *why* it doesn't match your profile. Sub-second response. Works in-store or at home in your pantry.

**The "Fig" Profile (their version of a user diet fingerprint)** Supports 2,800+ dietary restrictions and allergies — Low FODMAP, Gluten-Free, Vegan, Low Histamine, Alpha-Gal, and down to specific ingredient avoidances like walnut or corn allergy. The profile is the entire engine. Everything is filtered through it.

**Product Discovery by Store** Browse curated lists of safe products at 100+ grocery chains — so you can browse *before* you even go shopping. You see what's available to you at your specific local store.

**Multiple Profiles (Fig+ / paid)** Make a profile for everyone you care about and find food that works for everyone at once. Big hit with parents juggling multiple kids with different allergies.

**Restaurant Mode (Fig+ / paid)** Shows items that are likely safe at popular restaurants. Even tells you how to modify orders to make them work for your diet.

**Shopping Lists** Create shopping lists and save hours at the grocery store.

**Ingredient Education** Learn about ingredients and follow complex diets with confidence — it doesn't just say "nope", it tells you what the ingredient is and why it conflicts.

**Free vs Fig+ (paid)** Free: barcode scanning (limited scans), product discovery, one profile. Paid unlocks: unlimited scans, multiple profiles, restaurants, shopping lists.

***

## The Dirty Secret (Worth Knowing for Joule)

Fig relies on generic product databases — basically repositories of what's printed on the label, including voluntary precautionary allergen warnings like "may contain traces of milk." The FDA doesn't mandate these PAL warnings, so apps relying on them miss cross-contact risks entirely. They found products Fig cleared as safe that were being manufactured on shared lines with peanuts, soy, wheat — stuff that would kill someone with a severe allergy.

This is actually your opportunity with Joule. Since you're already using gpt 4-o as your multimodal model, you could have it *actually read the label from a photo* rather than just looking up a barcode in a database. That's genuinely better than what Fig does, and a real differentiator.

***

## Joule Integration Ideas

* **Barcode → Open Food Facts / USDA FoodData API** for the product lookup (both free, massive databases)
* **Camera scan of the label** → gpt 4-o reads the actual ingredients list directly, no database dependency
* **User dietary profile** feeds into whether ingredients flag or pass
* **"Why is this flagged?"** explanation per ingredient — users love that in Fig
* **Confidence indicator** so you're not misleading people the way Fig does

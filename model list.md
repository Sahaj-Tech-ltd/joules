# model list

Here's the full table. Starred ones (⭐) are the best budget picks for Joule's use case — high-volume food scanning, ingredient parsing, cheap multimodal calls.

***

## Anthropic — Claude API Models

| Model              | Input (\$/1M) | Output (\$/1M) | Context | Vision | Notes                      |
| ------------------ | ------------- | -------------- | ------- | ------ | -------------------------- |
| Claude Opus 4.6    | \$5.00        | \$25.00        | 1M      | ✅      | Flagship, best reasoning   |
| Claude Sonnet 4.6  | \$3.00        | \$15.00        | 1M      | ✅      | Best balance, recommended  |
| Claude Sonnet 4.5  | \$3.00        | \$15.00        | 1M      | ✅      | Same price, slightly older |
| ⭐ Claude Haiku 4.5 | \$1.00        | \$5.00         | 200K    | ✅      | Fast, cheap, solid quality |
| Claude Opus 4.5    | \$5.00        | \$25.00        | 1M      | ✅      | Same tier as 4.6           |
| Claude Opus 4.1    | \$15.00       | \$75.00        | 200K    | ✅      | Legacy, expensive, skip it |
| Claude Sonnet 4    | \$3.00        | \$15.00        | 200K    | ✅      | Legacy                     |
| Claude Opus 4      | \$15.00       | \$75.00        | 200K    | ✅      | Legacy, very expensive     |
| ⭐ Claude Haiku 3   | \$0.25        | \$1.25         | 200K    | ✅      | Cheapest Anthropic model   |

***

## OpenAI — API Models

| Model          | Input (\$/1M) | Output (\$/1M) | Context | Vision | Notes                                        |
| -------------- | ------------- | -------------- | ------- | ------ | -------------------------------------------- |
| GPT-5.4        | \$2.50        | \$15.00        | 1M      | ✅      | Current flagship                             |
| ⭐ GPT-5.4 mini | \~\$0.40      | \~\$1.60       | 128K    | ✅      | Strong mini, great for structured tasks      |
| ⭐ GPT-5.4 nano | \~\$0.10      | \~\$0.40       | 32K     | ✅      | Cheapest GPT-5.x family                      |
| GPT-5.4 pro    | \~\$10.50     | \~\$84.00      | 1M      | ✅      | Premium reasoning                            |
| GPT-4.1        | \$2.00        | \$8.00         | 1M      | ✅      | Long context champ                           |
| ⭐ GPT-4.1 mini | \~\$0.40      | \~\$1.60       | 1M      | ✅      | Instruction-following beast, 1M ctx          |
| ⭐ GPT-4.1 nano | \$0.10        | \$0.40         | 128K    | ✅      | Dirt cheap, great for routing/classification |
| ⭐ GPT-4o mini  | \$0.15        | \$0.60         | 128K    | ✅      | Proven workhorse, very reliable              |
| GPT-4o         | \$2.50        | \$10.00        | 128K    | ✅      | Previous gen flagship                        |
| o4-mini        | \~\$1.10      | \~\$4.40       | 200K    | ✅      | Reasoning, good price/perf                   |
| o3             | \$2.00        | \$8.00         | 200K    | ✅      | Strong reasoner                              |

***

## Joule Recommendation

For your use case (label vision scanning, ingredient parsing, diet matching), the sweet spot is:

* **Primary**: `GPT-4.1 mini` or `GPT-4o mini` — proven multimodal, dirt cheap, fast, handles label images well
* **Fallback/routing**: `GPT-4.1 nano` or `Claude Haiku 3` for simple text classification like "does this ingredient match user's profile" — literally fractions of a cent per call

The barcode lookup itself costs \$0, just a database hit. The AI cost only kicks in when you do camera-based label reading or ingredient explanation, which is where Qwen3-VL shines anyway. You may not even need OpenAI/Anthropic for the scanning part at all — just for the "why is this bad for me?" explanation layer.



add different things like mentioned here, fallback / routing, for using diff models. 

much like what we have for ocr model 

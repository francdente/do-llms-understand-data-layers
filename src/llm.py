import os
from dataclasses import dataclass

from litellm import completion  # type: ignore


@dataclass
class LLMConfig:
    model: str
    api_key: str
    base_url: str | None = None


@dataclass
class LLMResponse:
    content: str | None
    reasoning: str | None
    finish_reason: str | None
    usage: dict | None


def invoke_llm(
    prompt: str,
    model: str,
    api_key: str,
    base_url: str | None = None,
    temperature: float = 0.0,
    reasoning_effort: str | None = None,
) -> LLMResponse:
    disable_thinking = os.environ.get("LLM_DISABLE_THINKING", "").lower() in ("1", "true")

    if disable_thinking and model.startswith("ollama"):
        prompt = prompt + "\n/no_think"

    # reasoning models (e.g. GPT-5) require temperature=1 when reasoning is enabled
    if reasoning_effort and reasoning_effort != "none":
        temperature = 1.0

    kwargs = dict(
        model=model,
        messages=[{"role": "user", "content": prompt}],
        temperature=temperature,
        api_key=api_key,
        max_tokens=int(os.environ.get("LLM_MAX_TOKENS", 4096)),
    )

    if reasoning_effort:
        kwargs["reasoning_effort"] = reasoning_effort

    if base_url:
        kwargs["api_base"] = base_url
    if disable_thinking:
        if model.startswith("together_ai"):
            kwargs["extra_body"] = {"reasoning": {"enabled": False}}  # type: ignore
        elif not model.startswith(("ollama", "openai")):
            # chat_template_kwargs only works for vLLM/TGI-hosted models
            kwargs["extra_body"] = {"chat_template_kwargs": {"enable_thinking": False}}  # type: ignore

    response = completion(**kwargs, num_retries=3)  # type: ignore
    choice = response.choices[0]
    message = choice.message

    reasoning = getattr(message, "reasoning_content", None) or (message.model_dump().get("provider_specific_fields") or {}).get("reasoning_content")

    usage = None
    if response.usage:
        usage = {
            "prompt_tokens": response.usage.prompt_tokens,
            "completion_tokens": response.usage.completion_tokens,
            "total_tokens": response.usage.total_tokens,
        }

    return LLMResponse(
        content=message.content,
        reasoning=reasoning,
        finish_reason=choice.finish_reason,
        usage=usage,
    )

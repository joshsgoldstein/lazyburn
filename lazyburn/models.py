from dataclasses import dataclass, field
from datetime import datetime
from typing import Optional


@dataclass
class TokenUsage:
    input: int = 0
    cache_write_5m: int = 0
    cache_write_1h: int = 0
    cache_read: int = 0
    output: int = 0
    cost: float = 0.0

    def __iadd__(self, other: "TokenUsage") -> "TokenUsage":
        self.input += other.input
        self.cache_write_5m += other.cache_write_5m
        self.cache_write_1h += other.cache_write_1h
        self.cache_read += other.cache_read
        self.output += other.output
        self.cost += other.cost
        return self

    @property
    def cache_write(self) -> int:
        return self.cache_write_5m + self.cache_write_1h

    @property
    def total(self) -> int:
        return self.input + self.cache_write + self.cache_read + self.output


@dataclass
class Session:
    session_id: str
    project_path: str
    slug: str = ""
    start_time: Optional[datetime] = None
    end_time: Optional[datetime] = None
    usage: TokenUsage = field(default_factory=TokenUsage)
    models: set = field(default_factory=set)
    turn_count: int = 0
    last_prompt: str = ""

    @property
    def duration_minutes(self) -> float:
        if self.start_time and self.end_time:
            return (self.end_time - self.start_time).total_seconds() / 60
        return 0.0

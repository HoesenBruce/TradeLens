"""Disposable synthetic B design proof; NOT an application parser or security gate.

Run: python3 docs/validation/issue300/contract_sketch.py
No network, database, provider configuration or third-party packages.
"""
import copy
import json
import unittest

CATALOG = {
    "define_stop": ("risk", "warn", "record_stop", "stop_recorded", bool, lambda v: v is False),
    "review_loss_streak": ("session", "warn", "pause_review", "loss_streak", int, lambda v: 2 <= v <= 10000),
    "review_execution": ("execution", "tip", "review_journal", "execution_grade", int, lambda v: 1 <= v <= 5),
    "record_review": ("process", "tip", "grade_trade", "review_grade_absent", bool, lambda v: v is True),
}
KEYS = {"schema_version", "template_id", "category", "priority", "fact_refs", "action", "params"}
BINDING = ("synthetic-user", "synthetic-account", "synthetic-trade", "synthetic-run")


def require(condition):
    if not condition:
        raise ValueError("contract_rejected")


def pairs(items):
    result = {}
    for key, value in items:
        require(key not in result)
        result[key] = value
    return result


def decode(raw, limit=4096):
    require(len(raw) <= limit)
    def invalid_constant(_):
        raise ValueError("contract_rejected")
    obj = json.loads(raw.decode("utf-8"), object_pairs_hook=pairs, parse_constant=invalid_constant)
    def check(value, depth=0):
        if isinstance(value, str):
            value.encode("utf-8", errors="strict")
        if isinstance(value, (list, dict)):
            require(depth < 4)
            for child in (list(value.keys()) + list(value.values()) if isinstance(value, dict) else value):
                check(child, depth + 1)
    check(obj)
    require(type(obj) is dict and type(obj.get("schema_version")) is int and obj["schema_version"] == 1)
    return obj


def validate(obj, facts, seen):
    require(set(obj) == KEYS)
    require(type(obj["schema_version"]) is int and obj["schema_version"] == 1)
    template = obj["template_id"]
    require(type(template) is str and template in CATALOG and template not in seen)
    category, priority, action, kind, value_type, predicate = CATALOG[template]
    require((obj["category"], obj["priority"], obj["action"]) == (category, priority, action))
    require(type(obj["params"]) is dict and obj["params"] == {})
    refs = obj["fact_refs"]
    require(type(refs) is list and len(refs) == 1 and type(refs[0]) is str)
    require(refs[0] in facts)
    binding, fact_kind, value = facts[refs[0]]
    require(binding == BINDING and fact_kind == kind and type(value) is value_type and predicate(value))
    require(not ({template} | seen >= {"record_review", "review_execution"}))
    require(len(seen) < 4)
    seen.add(template)
    return template


def stream(records, facts, normal_finish=True, save_ok=True):
    """Iterator illustrates pre-end emission only; no real socket or transaction."""
    seen, ended, total = set(), False, 0
    try:
        for raw in records:
            total += len(raw)
            require(total <= 32768 and not ended)
            obj = decode(raw)
            if set(obj) == {"schema_version", "kind"} and obj["kind"] == "end":
                ended = True
            else:
                # Output a catalog ID, never a provider string to render.
                yield ("finding", validate(obj, facts, seen))
        require(ended and normal_finish and save_ok)
        yield ("completed", None)
    except (ValueError, UnicodeError, TypeError):
        yield ("error", "generation_failed")


def record(template="define_stop", ref="f0"):
    category, priority, action, *_ = CATALOG[template]
    return dict(schema_version=1, template_id=template, category=category,
                priority=priority, fact_refs=[ref], action=action, params={})


def encoded(value):
    return json.dumps(value, separators=(",", ":")).encode()


END = b'{"schema_version":1,"kind":"end"}'
FACTS = {"f0": (BINDING, "stop_recorded", False)}


class ContractProof(unittest.TestCase):
    def test_all_templates(self):
        for template, value in zip(CATALOG, [False, 2, 5, True]):
            with self.subTest(template=template):
                facts = {"f0": (BINDING, CATALOG[template][3], value)}
                self.assertEqual(list(stream([encoded(record(template)), END], facts))[-1][0], "completed")

    def test_schema_attacks(self):
        base = record()
        variants = []
        for key in base:
            missing = base.copy()
            del missing[key]
            variants.extend([missing, dict(base, **{key: None})])
        for key, values in {
            "schema_version": [True, 1.0, "1", 2],
            "template_id": ["foreign", "define_stop ", [], "SECRET"],
            "category": ["process"], "priority": ["neg"], "action": ["SECRET"],
            "params": [{"text": "SECRET"}, {"amount": 42}, []],
            "fact_refs": [[], ["f0", "f0"], ["foreign"], [5], "f0"],
        }.items():
            variants.extend(dict(base, **{key: value}) for value in values)
        variants.extend([dict(base, headline="SECRET"), [], dict(base, model="SECRET")])
        for index, value in enumerate(variants):
            with self.subTest(case=index):
                self.assertEqual(list(stream([encoded(value), END], FACTS)), [("error", "generation_failed")])

    def test_decoder_attacks(self):
        valid = encoded(record())
        attacks = [valid[:-1], valid + b'{}', b'\xff', valid + b' ' * 4096,
                   valid.replace(b'"schema_version":1', b'"schema_version":1,"schema_version":1'),
                   valid.replace(b'"schema_version":1', b'"schema_version":1,"\\u0073chema_version":1'),
                   valid.replace(b'"params":{}', b'"params":{"x":1,"x":2}'),
                   valid.replace(b'"params":{}', b'"params":{"x":[[[[]]]]}'),
                   valid.replace(b'"define_stop"', b'"\\ud800"'),
                   valid.replace(b'"schema_version":1', b'"schema_version":NaN'),
                   valid.replace(b'"schema_version":1', b'"schema_version":1e0')]
        for index, raw in enumerate(attacks):
            with self.subTest(case=index):
                self.assertEqual(list(stream([raw, END], FACTS)), [("error", "generation_failed")])

    def test_reference_binding(self):
        for index in range(4):
            binding = list(BINDING)
            binding[index] = "foreign"
            facts = {"f0": (tuple(binding), "stop_recorded", False)}
            self.assertEqual(list(stream([encoded(record()), END], facts))[0][0], "error")
        for facts in [{}, {"f0": (BINDING, "stop_recorded", None)},
                      {"f0": (BINDING, "stop_recorded", 0)},
                      {"f0": (BINDING, "stop_recorded", True)},
                      {"f0": (BINDING, "other", False)}]:
            self.assertEqual(list(stream([encoded(record()), END], facts))[0][0], "error")

    def test_numeric_domains(self):
        for template, values in [("review_execution", [0, 6, True, 1.5]),
                                 ("review_loss_streak", [-1, 1, 10001, True])]:
            for value in values:
                facts = {"f0": (BINDING, CATALOG[template][3], value)}
                self.assertEqual(list(stream([encoded(record(template)), END], facts))[0][0], "error")

    def test_incremental_and_late_failure(self):
        released = False
        def provider():
            yield encoded(record())
            require(released)
            yield END
        events = stream(provider(), FACTS)
        self.assertEqual(next(events), ("finding", "define_stop"))
        released = True
        self.assertEqual(next(events), ("completed", None))
        for tail in [[], [b'{"headline":"SECRET"}'], [encoded(record())], [END, END], [END, b'{}']]:
            result = list(stream([encoded(record())] + tail, FACTS))
            self.assertEqual(result[0], ("finding", "define_stop"))
            self.assertEqual(result[-1][0], "error")
            self.assertNotIn("completed", [event[0] for event in result])

    def test_terminal_and_save(self):
        self.assertEqual(list(stream([END], FACTS)), [("completed", None)])
        for options in [dict(normal_finish=False), dict(save_ok=False)]:
            self.assertEqual(list(stream([encoded(record()), END], FACTS, **options))[-1][0], "error")

    def test_contradictory_snapshot(self):
        facts = copy.deepcopy(FACTS)
        facts.update(f1=(BINDING, "execution_grade", 3), f2=(BINDING, "review_grade_absent", True))
        rows = [encoded(record("review_execution", "f1")), encoded(record("record_review", "f2")), END]
        self.assertEqual(list(stream(rows, facts))[-1][0], "error")


if __name__ == "__main__":
    unittest.main(verbosity=2)

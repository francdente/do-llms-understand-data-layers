from src.eval import Count, Metrics


def _total_metrics(counts: list[Count]) -> Metrics:
    sum = Count()
    for count in counts:
        sum.add(count)
    return sum.to_metrics()

#FLASK LANGUAGE
expert_0_CD_short_P2_test = Count(fn=1)
expert_0_ER_short_N1_test = Count(tn=1)
expert_0_SA_short_P2_test = Count(tp=1)
expert_0 = _total_metrics([expert_0_CD_short_P2_test, expert_0_SA_short_P2_test])

expert_1_CD_short_P3_test = Count(fn=1)
expert_1_ER_short_N3_test = Count(tn=1)
expert_1_SA_short_P3_test = Count(fn=1)
expert_1 = _total_metrics([expert_1_CD_short_P3_test, expert_1_SA_short_P3_test])

expert_2_CD_short_N3_test = Count(tn=1)
expert_2_ER_short_P1_test = Count(fn=1)
expert_2_SA_short_N3_test = Count(tn=1)
expert_2 = _total_metrics([expert_2_ER_short_P1_test])

expert_3_CD_short_P1_test = Count(tp=1)
expert_3_ER_short_N2_test = Count(tn=1)
expert_3_SA_short_N1_test = Count(tn=1)
expert_3 = _total_metrics([expert_3_CD_short_P1_test])

expert_4_CD_short_N2_test = Count(tn=1)
expert_4_ER_short_P3_test = Count(tp=1)
expert_4_SA_short_P1_test = Count(fn=1)
expert_4 = _total_metrics([expert_4_ER_short_P3_test, expert_4_SA_short_P1_test])

expert_5_CD_short_N1_test = Count(tn=1)
expert_5_ER_short_P2_test = Count(tp=1)
expert_5_SA_short_N2_test = Count(tn=1)
expert_5 = _total_metrics([expert_5_ER_short_P2_test])

#Other languages (see in the variable name)

expert_6_CD_short_P1_test_fastapi_raw = Count(fn=1)
expert_6_SA_short_P2_test_fastapi_raw = Count(tp=1)
expert_6_ER_short_P3_test_fastapi_raw = Count(tp=1)
expert_6 = _total_metrics(
    [
        expert_6_CD_short_P1_test_fastapi_raw,
        expert_6_SA_short_P2_test_fastapi_raw,
        expert_6_ER_short_P3_test_fastapi_raw,
    ]
)

expert_7_CD_short_P1_test_express_raw = Count(fn=1)
expert_7_SA_short_P2_test_express_raw = Count(fn=1)
expert_7_ER_short_P3_test_express_raw = Count(fn=1)
expert_7 = _total_metrics(
    [
        expert_7_CD_short_P1_test_express_raw,
        expert_7_SA_short_P2_test_express_raw,
        expert_7_ER_short_P3_test_express_raw,
    ]
)

expert_8_CD_short_P2_test_fastapi_raw = Count(fn=1)
expert_8_SA_short_P3_test_fastapi_raw = Count(fn=1)
expert_8_ER_short_P1_test_fastapi_raw = Count(tp=1)
expert_8 = _total_metrics(
    [
        expert_8_CD_short_P2_test_fastapi_raw,
        expert_8_SA_short_P3_test_fastapi_raw,
        expert_8_ER_short_P1_test_fastapi_raw,
    ]
)

expert_9_CD_short_P2_test_express_raw = Count(tp=1)
expert_9_SA_short_P3_test_express_raw = Count(fn=1)
expert_9_ER_short_P1_test_express_raw = Count(fn=1)
expert_9 = _total_metrics(
    [
        expert_9_CD_short_P2_test_express_raw,
        expert_9_SA_short_P3_test_express_raw,
        expert_9_ER_short_P1_test_express_raw,
    ]
)

expert_10_CD_short_P3_test_fastapi_raw = Count(fn=1)
expert_10_SA_short_P1_test_fastapi_raw = Count(fn=1)
expert_10_ER_short_P2_test_fastapi_raw = Count(tp=1)
expert_10 = _total_metrics(
    [
        expert_10_CD_short_P3_test_fastapi_raw,
        expert_10_SA_short_P1_test_fastapi_raw,
        expert_10_ER_short_P2_test_fastapi_raw,
    ]
)

expert_11_CD_short_P3_test_express_raw = Count(tp=1)
expert_11_SA_short_P1_test_express_raw = Count(tp=1)
expert_11_ER_short_P2_test_express_raw = Count(tp=1)
expert_11 = _total_metrics(
    [
        expert_11_CD_short_P3_test_express_raw,
        expert_11_SA_short_P1_test_express_raw,
        expert_11_ER_short_P2_test_express_raw,
    ]
)

expert_12_CD_short_P1_test_fastify_raw = Count(fn=1)
expert_12_SA_short_P2_test_fastify_raw = Count(fn=1)
expert_12_ER_short_P3_test_fastify_raw = Count(tp=1)
expert_12 = _total_metrics(
    [
        expert_12_CD_short_P1_test_fastify_raw,
        expert_12_SA_short_P2_test_fastify_raw,
        expert_12_ER_short_P3_test_fastify_raw,
    ]
)

expert_13_CD_short_P2_test_fastify_raw = Count(fn=1)
expert_13_SA_short_P3_test_fastify_raw = Count(fn=1)
expert_13_ER_short_P1_test_fastify_raw = Count(tp=1)
expert_13 = _total_metrics(
    [
        expert_13_CD_short_P2_test_fastify_raw,
        expert_13_SA_short_P3_test_fastify_raw,
        expert_13_ER_short_P1_test_fastify_raw,
    ]
)

expert_14_CD_short_P3_test_fastify_raw = Count(fn=1)
expert_14_SA_short_P1_test_fastify_raw = Count(fn=1)
expert_14_ER_short_P2_test_fastify_raw = Count(fn=1)
expert_14 = _total_metrics(
    [
        expert_14_CD_short_P3_test_fastify_raw,
        expert_14_SA_short_P1_test_fastify_raw,
        expert_14_ER_short_P2_test_fastify_raw,
    ]
)

expert_15_CD_short_P1_test_go_gin_raw = Count(fn=1)
expert_15_SA_short_P2_test_go_gin_raw = Count(tp=1)
expert_15_ER_short_P3_test_go_gin_raw = Count(tp=1)
expert_15 = _total_metrics(
    [
        expert_15_CD_short_P1_test_go_gin_raw,
        expert_15_SA_short_P2_test_go_gin_raw,
        expert_15_ER_short_P3_test_go_gin_raw,
    ]
)

by_label = {
    "cascade_delete": _total_metrics(
        [
            expert_0_CD_short_P2_test,
            expert_1_CD_short_P3_test,
            # expert_2_CD_short_N3_test,
            expert_3_CD_short_P1_test,
            # expert_4_CD_short_N2_test,
            # expert_5_CD_short_N1_test,
            expert_6_CD_short_P1_test_fastapi_raw,
            expert_7_CD_short_P1_test_express_raw,
            expert_8_CD_short_P2_test_fastapi_raw,
            expert_9_CD_short_P2_test_express_raw,
            expert_10_CD_short_P3_test_fastapi_raw,
            expert_11_CD_short_P3_test_express_raw,
            expert_12_CD_short_P1_test_fastify_raw,
            expert_13_CD_short_P2_test_fastify_raw,
            expert_14_CD_short_P3_test_fastify_raw,
            expert_15_CD_short_P1_test_go_gin_raw,
        ]
    ),
    "stale_aggregate": _total_metrics(
        [
            expert_0_SA_short_P2_test,
            expert_1_SA_short_P3_test,
            # expert_2_SA_short_N3_test,
            # expert_3_SA_short_N1_test,
            expert_4_SA_short_P1_test,
            # expert_5_SA_short_N2_test,
            expert_6_SA_short_P2_test_fastapi_raw,
            expert_7_SA_short_P2_test_express_raw,
            expert_8_SA_short_P3_test_fastapi_raw,
            expert_9_SA_short_P3_test_express_raw,
            expert_10_SA_short_P1_test_fastapi_raw,
            expert_11_SA_short_P1_test_express_raw,
            expert_12_SA_short_P2_test_fastify_raw,
            expert_13_SA_short_P3_test_fastify_raw,
            expert_14_SA_short_P1_test_fastify_raw,
            expert_15_SA_short_P2_test_go_gin_raw,
        ]
    ),
    "exposed_record": _total_metrics(
        [
            # expert_0_ER_short_N1_test,
            # expert_1_ER_short_N3_test,
            expert_2_ER_short_P1_test,
            # expert_3_ER_short_N2_test,
            expert_4_ER_short_P3_test,
            expert_5_ER_short_P2_test,
            expert_6_ER_short_P3_test_fastapi_raw,
            expert_7_ER_short_P3_test_express_raw,
            expert_8_ER_short_P1_test_fastapi_raw,
            expert_9_ER_short_P1_test_express_raw,
            expert_10_ER_short_P2_test_fastapi_raw,
            expert_11_ER_short_P2_test_express_raw,
            expert_12_ER_short_P3_test_fastify_raw,
            expert_13_ER_short_P1_test_fastify_raw,
            expert_14_ER_short_P2_test_fastify_raw,
            expert_15_ER_short_P3_test_go_gin_raw,
        ]
    ),
}

print("== AGGREGATE BY EXPERT")
print(f"\n# expert_0\n{expert_0}")
print(f"\n# expert_1:\n{expert_1}")
print(f"\n# expert_2\n{expert_2}")
print(f"\n# expert_3\n{expert_3}")
print(f"\n# expert_4\n{expert_4}")
print(f"\n# expert_5\n{expert_5}")

print(f"\n# expert_6\n{expert_6}")
print(f"\n# expert_7\n{expert_7}")
print(f"\n# expert_8\n{expert_8}")
print(f"\n# expert_9\n{expert_9}")
print(f"\n# expert_10\n{expert_10}")
print(f"\n# expert_11\n{expert_11}")
print(f"\n# expert_12\n{expert_12}")
print(f"\n# expert_13\n{expert_13}")
print(f"\n# expert_14\n{expert_14}")
print(f"\n# expert_15\n{expert_15}")


print("\n\n== AGGREGATE BY LABEL")
print(f"\n# cascade_delete\n{by_label['cascade_delete']}")
print(f"\n# stale_aggregate\n{by_label['stale_aggregate']}")
print(f"\n# exposed_record\n{by_label['exposed_record']}")

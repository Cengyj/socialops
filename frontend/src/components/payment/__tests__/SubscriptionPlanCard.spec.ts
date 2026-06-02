import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import { createI18n } from "vue-i18n";
import SubscriptionPlanCard from "../SubscriptionPlanCard.vue";
import type { SubscriptionPlan } from "@/types/payment";

const i18n = createI18n({
  legacy: false,
  locale: "en",
  fallbackWarn: false,
  missingWarn: false,
  messages: {
    en: {
      payment: {
        days: "days",
        planCard: {
          quota: "Quota",
          rate: "Rate",
          unlimited: "Unlimited",
        },
        subscribeNow: "Subscribe now",
      },
    },
  },
});

const mountPlanCard = (groupPlatform: string) =>
  mount(SubscriptionPlanCard, {
    props: {
      plan: {
        id: 1,
        group_id: 10,
        group_platform: groupPlatform,
        name: "Pro",
        price: 10,
        amount: 1000,
        features: [],
        rate_multiplier: 1,
        validity_days: 30,
        validity_unit: "day",
        legacy_scope_labels: ["legacy-scope-one", "legacy-scope-two"],
        is_active: true,
      } as unknown as SubscriptionPlan,
    },
    global: { plugins: [i18n] },
  });

describe("SubscriptionPlanCard", () => {
  it("does not show legacy scope labels for social plans", () => {
    const text = mountPlanCard("x_twitter").text();

    expect(text).not.toContain("legacy-scope-one");
    expect(text).not.toContain("legacy-scope-two");
  });

  it("falls back to the generic social badge for unknown legacy plan families", () => {
    const text = mountPlanCard("legacy").text();

    expect(text).toContain("Pro");
    expect(text).not.toContain("legacy-scope-one");
  });
});

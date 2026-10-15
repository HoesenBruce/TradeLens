import { createFileRoute } from "@tanstack/react-router";
import { ReviewInboxView } from "@/app/screens/ReviewInboxView";
export const Route = createFileRoute("/review")({ component: ReviewInboxView });

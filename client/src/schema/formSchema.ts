import { z, ZodIssueCode } from 'zod';

export const formSchema = z.object({
    username: z.string().optional(),
    password: z.string(),
}).superRefine((data, ctx) => {
    if (data.username && data.username !== 'guest' && data.password.length < 6) {
        ctx.addIssue({
            code: ZodIssueCode.custom,
            path: ["password"],
            message: "Password must be at least 6 characters.",
        });
    }
});

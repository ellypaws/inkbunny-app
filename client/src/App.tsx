import React from 'react';
import {useForm} from 'react-hook-form';
import {z} from 'zod';
import {zodResolver} from '@hookform/resolvers/zod';
import Cookies from 'js-cookie';
import {
    Card,
    CardContent,
    CardDescription,
    CardFooter,
    CardHeader,
    CardTitle,
} from '@/components/ui/card'; // Adjust the import path according to your project structure
import {Button} from "@/components/ui/button";
import {Input} from '@/components/ui/input'; // Adjust the import path according to your project structure
import {Form, FormControl, FormField, FormItem, FormLabel, FormMessage} from '@/components/ui/form';

const formSchema = z.object({
    username: z.string().min(2, {message: "Username must be at least 2 characters."}),
    password: z.string().min(6, {message: "Password must be at least 6 characters."}),
});

const LoginForm = () => {
    const form = useForm({
        resolver: zodResolver(formSchema),
        defaultValues: {
            username: '',
            password: '',
        },
    });

    const onSubmit = async (values) => {
        const response = await fetch('http://localhost:1323/api/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(values),
        });

        if (response.ok) {
            const data = await response.json();
            Cookies.set('PHPSESSID', data.sessionId);
            alert('Login successful!');
            // Navigate to the next page or update UI accordingly
        } else {
            alert('Login failed!');
            // Handle login failure
        }
    };

    return (
        <Card className="w-[350px]">
            <CardHeader>
                <CardTitle>Login</CardTitle>
                <CardDescription>Access your account.</CardDescription>
            </CardHeader>
            <CardContent>
                <Form {...form}>
                    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-8">
                        <FormField
                            control={form.control}
                            name="username"
                            render={({field}) => (
                                <FormItem>
                                    <FormLabel>Username</FormLabel>
                                    <FormControl>
                                        <Input placeholder="guest" {...field} />
                                    </FormControl>
                                    <FormMessage/>
                                </FormItem>
                            )}
                        />
                        <FormField
                            control={form.control}
                            name="password"
                            render={({field}) => (
                                <FormItem>
                                    <FormLabel>Password</FormLabel>
                                    <FormControl>
                                        <Input type="password" placeholder="Password" {...field} />
                                    </FormControl>
                                    <FormMessage/>
                                </FormItem>
                            )}
                        />
                    </form>
                </Form>
            </CardContent>
            <CardFooter className="flex justify-end">
                <Button type="submit" onClick={form.handleSubmit(onSubmit)}>Login</Button>
            </CardFooter>
        </Card>
    );
};

export function App() {
    return (
        <div className="flex items-center justify-center h-screen">
            <LoginForm/>
        </div>
    );
}

export default App;
